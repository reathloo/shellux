package doctor

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/reathloo/shellux/internal/config"
	"github.com/reathloo/shellux/internal/themepack"
	"github.com/reathloo/shellux/shell"
)

func TestEmbeddedIntegrationVersionMatchesScripts(t *testing.T) {
	want := []byte("shellux_integration_version=" + strconv.Itoa(shell.IntegrationVersion))
	if !bytes.Contains(shell.Zsh, want) || !bytes.Contains(shell.Bash, want) {
		t.Fatalf("embedded shell integration version does not match %d", shell.IntegrationVersion)
	}
}

func TestIntegrationRepairOnlyReplacesInstalledRegularFiles(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	files, err := integrationFiles()
	if err != nil {
		t.Fatal(err)
	}
	if check := scanIntegration(); check.State != "warning" || !check.Repairable {
		t.Fatalf("missing integration = %+v", check)
	}
	if err := repairIntegration(context.Background()); err != nil {
		t.Fatal(err)
	}
	for path, want := range files {
		got, err := os.ReadFile(path)
		if err != nil || !bytes.Equal(got, want) {
			t.Fatalf("integration %s not restored: %v", path, err)
		}
	}
	if check := scanIntegration(); check.State != "ok" {
		t.Fatalf("repaired integration = %+v", check)
	}
	var path string
	for candidate := range files {
		path = candidate
		break
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(t.TempDir(), "untouched")
	if err := os.WriteFile(target, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, path); err != nil {
		t.Fatal(err)
	}
	if check := scanIntegration(); check.Repairable {
		t.Fatalf("symlink offered for repair: %+v", check)
	}
	if err := repairIntegration(context.Background()); err == nil {
		t.Fatal("symlink was replaced")
	}
	if got, _ := os.ReadFile(target); string(got) != "keep" {
		t.Fatalf("symlink target changed: %q", got)
	}
	t.Setenv("SHELLUX_DIAG_VERSION", "19")
	if check := scanIntegration(); !strings.Contains(check.Detail, "current shell is version 19") {
		t.Fatalf("old live shell not detected: %+v", check)
	}
}

func TestCorruptInstalledThemeIsDetected(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	entry, ok := themepack.Lookup("heart")
	if !ok {
		t.Fatal("missing fixture")
	}
	store, err := themepack.DefaultStore()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(store.Directory, 0700); err != nil {
		t.Fatal(err)
	}
	receipt, _ := json.Marshal(themepack.Catalog{Format: themepack.Format, Themes: []themepack.Entry{entry}})
	receiptPath := filepath.Join(store.Directory, entry.Name+".installed.json")
	if err := os.WriteFile(receiptPath, receipt, 0600); err != nil {
		t.Fatal(err)
	}
	check := scanThemes()
	if check.State != "warning" || !check.Repairable || !strings.Contains(check.Detail, entry.Name) {
		t.Fatalf("broken theme = %+v", check)
	}
	target := filepath.Join(t.TempDir(), "receipt")
	if err := os.Rename(receiptPath, target); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, receiptPath); err != nil {
		t.Fatal(err)
	}
	if check := scanThemes(); check.Repairable {
		t.Fatalf("symlinked theme offered for repair: %+v", check)
	}
}

func TestPermissionRepairStaysOnOwnedShelluxFile(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	path, err := config.DefaultPath()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("{}"), 0400); err != nil {
		t.Fatal(err)
	}
	if check := scanPermissions(); check.State != "warning" || !check.Repairable {
		t.Fatalf("permissions = %+v", check)
	}
	if err := repairPermissions(context.Background()); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm()&0600 != 0600 {
		t.Fatalf("permissions not repaired: %v %v", info, err)
	}
}

func TestWatcherParserRequiresOwnedShelluxAndDeadParent(t *testing.T) {
	output := strings.Join([]string{
		"101 1 501 /usr/local/bin/shellux --watch --in-place --parent-pid 900",
		"102 900 501 /usr/local/bin/shellux --watch --in-place --parent-pid 900",
		"103 1 502 /usr/local/bin/shellux --watch --in-place --parent-pid 901",
		"104 1 501 /bin/bash -c shellux --watch --in-place --parent-pid 902",
		"105 1 501 /usr/local/bin/shellux --once --parent-pid 903",
	}, "\n")
	orphans := parseWatchers(output, 501, func(pid int) bool { return pid == 900 })
	if len(orphans) != 0 {
		t.Fatalf("live parent considered orphan: %+v", orphans)
	}
	orphans = parseWatchers(output, 501, func(int) bool { return false })
	if len(orphans) != 2 || orphans[0].pid != 101 || orphans[1].pid != 102 {
		t.Fatalf("orphan detection = %+v", orphans)
	}
}
