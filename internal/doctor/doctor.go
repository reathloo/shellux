// Package doctor checks local Shellux installation health and offers explicit,
// narrowly scoped repairs. Scanning never writes files or accesses the network.
package doctor

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/reathloo/shellux/internal/config"
	"github.com/reathloo/shellux/internal/themepack"
	"github.com/reathloo/shellux/shell"
)

const (
	Integration = "integration"
	Themes      = "themes"
	Permissions = "permissions"
	Watchers    = "watchers"
)

type Check struct {
	Key        string
	State      string // ok, warning or unknown
	Detail     string
	Repairable bool
}

func Scan(ctx context.Context) []Check {
	return []Check{scanIntegration(), scanThemes(), scanPermissions(), scanWatchers(ctx)}
}

func Repair(ctx context.Context, key string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	switch key {
	case Integration:
		return repairIntegration(ctx)
	case Themes:
		return repairThemes(ctx)
	case Permissions:
		return repairPermissions(ctx)
	case Watchers:
		return repairWatchers(ctx)
	default:
		return fmt.Errorf("unknown doctor check %q", key)
	}
}

func integrationDir() (string, error) {
	base := os.Getenv("XDG_DATA_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(base, "shellux", "shell"), nil
}

func integrationFiles() (map[string][]byte, error) {
	dir, err := integrationDir()
	if err != nil {
		return nil, err
	}
	return map[string][]byte{
		filepath.Join(dir, "shellux.zsh"):  shell.Zsh,
		filepath.Join(dir, "shellux.bash"): shell.Bash,
	}, nil
}

func scanIntegration() Check {
	check := Check{Key: Integration, State: "ok", Detail: "Installed Zsh and Bash integration matches this binary."}
	files, err := integrationFiles()
	if err != nil {
		check.State, check.Detail = "unknown", err.Error()
		return check
	}
	var outdated []string
	blocked := false
	for path, want := range files {
		if info, err := os.Lstat(path); err == nil && !info.Mode().IsRegular() {
			blocked = true
		}
		got, err := os.ReadFile(path)
		if err != nil || !bytes.Equal(got, want) {
			outdated = append(outdated, filepath.Base(path))
		}
	}
	if len(outdated) > 0 {
		check.State, check.Detail, check.Repairable = "warning", "Missing or outdated: "+strings.Join(outdated, ", "), !blocked
	}
	if version := os.Getenv("SHELLUX_DIAG_VERSION"); version != "" && version != strconv.Itoa(shell.IntegrationVersion) {
		check.State = "warning"
		check.Detail += "; current shell is version " + version + " (open a new tab after repair)"
		check.Repairable = !blocked
	}
	return check
}

func repairIntegration(ctx context.Context) error {
	files, err := integrationFiles()
	if err != nil {
		return err
	}
	for path := range files {
		if info, err := os.Lstat(path); err == nil && !info.Mode().IsRegular() {
			return fmt.Errorf("refusing to replace non-regular integration file %s", path)
		} else if err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	for path, data := range files {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}
		if err := replaceFile(path, data, 0644); err != nil {
			return err
		}
	}
	return nil
}

func replaceFile(path string, data []byte, mode os.FileMode) error {
	temp, err := os.CreateTemp(filepath.Dir(path), ".shellux-doctor-*")
	if err != nil {
		return err
	}
	defer os.Remove(temp.Name())
	if _, err = temp.Write(data); err == nil {
		err = temp.Chmod(mode)
	}
	if err == nil {
		err = temp.Sync()
	}
	if closeErr := temp.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	return os.Rename(temp.Name(), path)
}

func damagedThemes() ([]string, error) {
	store, err := themepack.DefaultStore()
	if err != nil {
		return nil, err
	}
	paths, err := filepath.Glob(filepath.Join(store.Directory, "*.installed.json"))
	if err != nil {
		return nil, err
	}
	var damaged []string
	for _, path := range paths {
		name := strings.TrimSuffix(filepath.Base(path), ".installed.json")
		entry, err := store.Installed(name)
		if err == nil {
			_, err = store.Load(entry)
		}
		if err != nil {
			damaged = append(damaged, name)
		}
	}
	return damaged, nil
}

func scanThemes() Check {
	check := Check{Key: Themes, State: "ok", Detail: "Installed theme packages are valid."}
	damaged, err := damagedThemes()
	if err != nil {
		check.State, check.Detail = "unknown", err.Error()
		return check
	}
	if len(damaged) > 0 {
		store, storeErr := themepack.DefaultStore()
		if storeErr != nil {
			check.State, check.Detail = "unknown", storeErr.Error()
			return check
		}
		check.State, check.Detail = "warning", "Damaged: "+strings.Join(damaged, ", ")
		check.Repairable = true
		for _, name := range damaged {
			entry, ok := themepack.Lookup(name)
			if !ok || themePathUnsafe(store, entry) {
				check.Repairable = false
			}
		}
	}
	return check
}

func repairThemes(ctx context.Context) error {
	damaged, err := damagedThemes()
	if err != nil {
		return err
	}
	store, err := themepack.DefaultStore()
	if err != nil {
		return err
	}
	for _, name := range damaged {
		entry, ok := themepack.Lookup(name)
		if !ok {
			return fmt.Errorf("theme %q is not in the verified catalog", name)
		}
		if themePathUnsafe(store, entry) {
			return fmt.Errorf("theme %q uses a non-regular path; repair it manually", name)
		}
		if _, err := store.Install(ctx, entry); err != nil {
			return fmt.Errorf("redownload %s: %w", name, err)
		}
	}
	return nil
}

func themePathUnsafe(store *themepack.Store, entry themepack.Entry) bool {
	for _, path := range []string{
		filepath.Join(store.Directory, entry.Name+".installed.json"),
		filepath.Join(store.Directory, entry.Name+"-"+entry.Version+".shellux-theme"),
	} {
		info, err := os.Lstat(path)
		if err == nil && !info.Mode().IsRegular() {
			return true
		}
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return true
		}
	}
	return false
}

type permissionTarget struct {
	path      string
	needed    os.FileMode
	directory bool
}

func permissionTargets() ([]permissionTarget, error) {
	configPath, err := config.DefaultPath()
	if err != nil {
		return nil, err
	}
	store, err := themepack.DefaultStore()
	if err != nil {
		return nil, err
	}
	shellDir, err := integrationDir()
	if err != nil {
		return nil, err
	}
	return []permissionTarget{
		{configPath, 0600, false},
		{filepath.Dir(configPath), 0700, true},
		{store.Directory, 0700, true},
		{shellDir, 0700, true},
		{filepath.Join(shellDir, "shellux.zsh"), 0600, false},
		{filepath.Join(shellDir, "shellux.bash"), 0600, false},
	}, nil
}

func permissionProblems() ([]permissionTarget, bool, error) {
	targets, err := permissionTargets()
	if err != nil {
		return nil, false, err
	}
	var problems []permissionTarget
	repairable := true
	for _, target := range targets {
		info, err := os.Lstat(target.path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, false, err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			continue
		} // Existing config symlinks are valid; never chmod through them.
		if target.directory != info.IsDir() {
			continue
		}
		if info.Mode().Perm()&target.needed != target.needed {
			problems = append(problems, target)
			if !ownedByUser(info) {
				repairable = false
			}
		}
	}
	return problems, repairable, nil
}

func scanPermissions() Check {
	check := Check{Key: Permissions, State: "ok", Detail: "Shellux files have owner read/write access."}
	problems, repairable, err := permissionProblems()
	if err != nil {
		check.State, check.Detail = "unknown", err.Error()
		return check
	}
	if len(problems) > 0 {
		paths := make([]string, 0, len(problems))
		for _, target := range problems {
			paths = append(paths, target.path)
		}
		check.State, check.Detail, check.Repairable = "warning", "Insufficient owner permissions: "+strings.Join(paths, ", "), repairable
	}
	return check
}

func repairPermissions(ctx context.Context) error {
	problems, repairable, err := permissionProblems()
	if err != nil {
		return err
	}
	if !repairable {
		return errors.New("some files are not owned by this user; repair them manually")
	}
	for _, target := range problems {
		if err := ctx.Err(); err != nil {
			return err
		}
		info, err := os.Lstat(target.path)
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 || !ownedByUser(info) {
			return fmt.Errorf("permissions changed while checking %s", target.path)
		}
		if err := os.Chmod(target.path, info.Mode().Perm()|target.needed); err != nil {
			return err
		}
	}
	return nil
}

type watcher struct {
	pid, parent int
	command     string
}

func orphanWatchers(ctx context.Context) ([]watcher, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "ps", "-eo", "pid=,ppid=,uid=,args=")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	return parseWatchers(string(output), os.Getuid(), processExists), nil
}

func parseWatchers(output string, uidWanted int, exists func(int) bool) []watcher {
	var result []watcher
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 7 {
			continue
		}
		pid, errPID := strconv.Atoi(fields[0])
		parent, errPPID := strconv.Atoi(fields[1])
		uid, errUID := strconv.Atoi(fields[2])
		if errPID != nil || errPPID != nil || errUID != nil || uid != uidWanted || filepath.Base(fields[3]) != "shellux" {
			continue
		}
		args := fields[4:]
		if !containsArg(args, "--watch") || !containsArg(args, "--in-place") {
			continue
		}
		declared := parentArg(args)
		if declared <= 1 || declared == pid {
			continue
		}
		if exists(declared) || (parent != 1 && parent != declared && exists(parent)) {
			continue
		}
		result = append(result, watcher{pid, declared, strings.Join(fields[3:], " ")})
	}
	return result
}

func containsArg(args []string, value string) bool {
	for _, arg := range args {
		if arg == value {
			return true
		}
	}
	return false
}

func parentArg(args []string) int {
	for i, arg := range args {
		if arg == "--parent-pid" && i+1 < len(args) {
			value, _ := strconv.Atoi(args[i+1])
			return value
		}
		if strings.HasPrefix(arg, "--parent-pid=") {
			value, _ := strconv.Atoi(strings.TrimPrefix(arg, "--parent-pid="))
			return value
		}
	}
	return 0
}

func processExists(pid int) bool {
	err := syscall.Kill(pid, 0)
	return err == nil || errors.Is(err, syscall.EPERM)
}

func scanWatchers(ctx context.Context) Check {
	check := Check{Key: Watchers, State: "ok", Detail: "No orphaned Shellux watchers found."}
	orphans, err := orphanWatchers(ctx)
	if err != nil {
		check.State, check.Detail = "unknown", err.Error()
		return check
	}
	if len(orphans) > 0 {
		pids := make([]string, 0, len(orphans))
		for _, process := range orphans {
			pids = append(pids, strconv.Itoa(process.pid))
		}
		check.State, check.Detail, check.Repairable = "warning", "Orphaned watcher PID(s): "+strings.Join(pids, ", "), true
	}
	return check
}

func repairWatchers(ctx context.Context) error {
	orphans, err := orphanWatchers(ctx)
	if err != nil {
		return err
	}
	for _, process := range orphans {
		if err := ctx.Err(); err != nil {
			return err
		}
		current, err := orphanWatchers(ctx)
		if err != nil {
			return err
		}
		for _, candidate := range current {
			if candidate.pid == process.pid && candidate.command == process.command {
				if err := syscall.Kill(process.pid, syscall.SIGTERM); err != nil && !errors.Is(err, syscall.ESRCH) {
					return err
				}
			}
		}
	}
	return nil
}
