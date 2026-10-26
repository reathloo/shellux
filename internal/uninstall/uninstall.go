// Package uninstall removes a user-level Shellux installation.
package uninstall

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	shelluxBlockStart = "# >>> shellux >>>"
	shelluxBlockEnd   = "# <<< shellux <<<"
	bashLoginStart    = "# >>> shellux bash login >>>"
	bashLoginEnd      = "# <<< shellux bash login <<<"
)

// Plan contains the exact user-level paths that may be changed or removed.
// Keeping the plan explicit makes the destructive operation testable.
type Plan struct {
	Executable       string
	DataDir          string
	InstallerDataDir string
	ConfigDir        string
	StateDir         string
	StartupFiles     []string
}

// DefaultPlan resolves the paths used by the Shellux installer and runtime.
func DefaultPlan() (Plan, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Plan{}, fmt.Errorf("find home directory: %w", err)
	}
	executable, err := os.Executable()
	if err != nil {
		return Plan{}, fmt.Errorf("find executable: %w", err)
	}
	if resolved, resolveErr := filepath.EvalSymlinks(executable); resolveErr == nil {
		executable = resolved
	}

	dataHome := os.Getenv("XDG_DATA_HOME")
	installerDataHome := dataHome
	if dataHome == "" {
		installerDataHome = filepath.Join(home, ".local", "share")
		dataHome = installerDataHome
		if runtime.GOOS == "darwin" {
			dataHome = filepath.Join(home, "Library", "Application Support")
		}
	}
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		configHome, err = os.UserConfigDir()
		if err != nil {
			return Plan{}, fmt.Errorf("find config directory: %w", err)
		}
	}
	stateHome := os.Getenv("XDG_STATE_HOME")
	if stateHome == "" {
		stateHome = filepath.Join(home, ".local", "state")
	}
	zshHome := os.Getenv("ZDOTDIR")
	if zshHome == "" {
		zshHome = home
	}

	return Plan{
		Executable:       executable,
		DataDir:          filepath.Join(dataHome, "shellux"),
		InstallerDataDir: filepath.Join(installerDataHome, "shellux"),
		ConfigDir:        filepath.Join(configHome, "shellux"),
		StateDir:         filepath.Join(stateHome, "shellux"),
		StartupFiles: []string{
			filepath.Join(zshHome, ".zshrc"),
			filepath.Join(home, ".zshrc"),
			filepath.Join(home, ".bashrc"),
			filepath.Join(home, ".bash_profile"),
			filepath.Join(home, ".bash_login"),
		},
	}, nil
}

// Run removes Shellux blocks from shell startup files, then deletes Shellux's
// private data and finally the currently running executable.
func (p Plan) Run() error {
	var failures []error
	seen := make(map[string]bool)
	for _, path := range p.StartupFiles {
		path = filepath.Clean(path)
		if seen[path] {
			continue
		}
		seen[path] = true
		if err := cleanStartupFile(path); err != nil {
			failures = append(failures, err)
		}
	}
	if len(failures) > 0 {
		return errors.Join(failures...)
	}
	for _, directory := range uniquePaths(p.ConfigDir, p.DataDir, p.InstallerDataDir, p.StateDir) {
		if filepath.Base(directory) != "shellux" {
			failures = append(failures, fmt.Errorf("refuse to remove unexpected directory %s", directory))
			continue
		}
		if err := os.RemoveAll(directory); err != nil {
			failures = append(failures, fmt.Errorf("remove %s: %w", directory, err))
		}
	}
	if len(failures) > 0 {
		return errors.Join(failures...)
	}
	if p.Executable != "" {
		if filepath.Base(p.Executable) != "shellux" {
			return fmt.Errorf("refuse to remove unexpected executable %s", p.Executable)
		}
		if err := os.Remove(p.Executable); err != nil && !errors.Is(err, os.ErrNotExist) {
			failures = append(failures, fmt.Errorf("remove executable: %w", err))
		}
	}
	return errors.Join(failures...)
}

func uniquePaths(paths ...string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(paths))
	for _, path := range paths {
		if path == "" {
			continue
		}
		path = filepath.Clean(path)
		if !seen[path] {
			seen[path] = true
			result = append(result, path)
		}
	}
	return result
}

func cleanStartupFile(path string) error {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect %s: %w", path, err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("refuse to edit non-regular startup file %s", path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	cleaned := removeShelluxLines(string(data))
	if cleaned == string(data) {
		return nil
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".shellux-uninstall-*")
	if err != nil {
		return fmt.Errorf("prepare %s: %w", path, err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(info.Mode().Perm()); err != nil {
		temporary.Close()
		return fmt.Errorf("set permissions for %s: %w", path, err)
	}
	if _, err := temporary.WriteString(cleaned); err != nil {
		temporary.Close()
		return fmt.Errorf("write %s: %w", path, err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close %s: %w", path, err)
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return fmt.Errorf("replace %s: %w", path, err)
	}
	return nil
}

func removeShelluxLines(content string) string {
	hadTrailingNewline := strings.HasSuffix(content, "\n")
	lines := strings.Split(strings.TrimSuffix(content, "\n"), "\n")
	result := make([]string, 0, len(lines))
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)
		end := ""
		if trimmed == shelluxBlockStart {
			end = shelluxBlockEnd
		} else if trimmed == bashLoginStart {
			end = bashLoginEnd
		}
		if end != "" {
			closing := -1
			for j := i + 1; j < len(lines); j++ {
				if strings.TrimSpace(lines[j]) == end {
					closing = j
					break
				}
			}
			if closing >= 0 {
				i = closing
				continue
			}
		}
		if isLegacyIntegrationLine(trimmed) {
			continue
		}
		result = append(result, line)
	}
	for len(result) > 0 && strings.TrimSpace(result[len(result)-1]) == "" {
		result = result[:len(result)-1]
	}
	cleaned := strings.Join(result, "\n")
	if hadTrailingNewline && cleaned != "" {
		cleaned += "\n"
	}
	return cleaned
}

func isLegacyIntegrationLine(line string) bool {
	if line == "" || strings.HasPrefix(line, "#") {
		return false
	}
	isSource := strings.HasPrefix(line, "source ") || strings.HasPrefix(line, ". ")
	return isSource && (strings.Contains(line, "shellux.zsh") || strings.Contains(line, "shellux.bash"))
}
