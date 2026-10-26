package uninstall

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunRemovesInstallationAndOnlyShelluxStartupEntries(t *testing.T) {
	root := t.TempDir()
	executable := filepath.Join(root, "prefix", "bin", "shellux")
	dataDir := filepath.Join(root, "data", "shellux")
	installerDataDir := filepath.Join(root, "installer-data", "shellux")
	configDir := filepath.Join(root, "config", "shellux")
	stateDir := filepath.Join(root, "state", "shellux")
	startup := filepath.Join(root, ".zshrc")
	writeTestFile(t, executable, "binary", 0755)
	writeTestFile(t, filepath.Join(dataDir, "themes", "purple.shellux-theme"), "theme", 0600)
	writeTestFile(t, filepath.Join(installerDataDir, "shell", "shellux.zsh"), "integration", 0600)
	writeTestFile(t, filepath.Join(configDir, "config.json"), "{}", 0600)
	writeTestFile(t, filepath.Join(stateDir, "update.json"), "{}", 0600)
	writeTestFile(t, startup, strings.Join([]string{
		"export KEEP=1",
		"# >>> shellux >>>",
		"export PATH='/tmp/prefix/bin':$PATH",
		"source '/tmp/data/shellux/shell/shellux.zsh'",
		"# <<< shellux <<<",
		"alias keep='printf ok'",
		"source \"$HOME/.local/share/shellux/shell/shellux.zsh\"",
		"",
	}, "\n"), 0640)

	plan := Plan{Executable: executable, DataDir: dataDir, InstallerDataDir: installerDataDir, ConfigDir: configDir, StateDir: stateDir, StartupFiles: []string{startup}}
	if err := plan.Run(); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{executable, dataDir, installerDataDir, configDir, stateDir} {
		if _, err := os.Lstat(path); !os.IsNotExist(err) {
			t.Fatalf("%s still exists", path)
		}
	}
	data, err := os.ReadFile(startup)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(data), "export KEEP=1\nalias keep='printf ok'\n"; got != want {
		t.Fatalf("startup file = %q, want %q", got, want)
	}
	if info, err := os.Stat(startup); err != nil || info.Mode().Perm() != 0640 {
		t.Fatalf("startup permissions changed: %v %v", info, err)
	}
}

func TestRunRefusesStartupSymlink(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "target")
	link := filepath.Join(root, ".bashrc")
	writeTestFile(t, target, "source /tmp/shellux.bash\n", 0600)
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if err := (Plan{StartupFiles: []string{link}}).Run(); err == nil || !strings.Contains(err.Error(), "non-regular") {
		t.Fatalf("Run() error = %v", err)
	}
	data, _ := os.ReadFile(target)
	if string(data) != "source /tmp/shellux.bash\n" {
		t.Fatal("symlink target was changed")
	}
}

func TestUnmatchedMarkerDoesNotRemoveFollowingLines(t *testing.T) {
	input := "# >>> shellux >>>\nexport KEEP=1\n"
	if got := removeShelluxLines(input); got != input {
		t.Fatalf("removeShelluxLines() = %q, want unchanged input", got)
	}
}

func writeTestFile(t *testing.T, path, content string, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		t.Fatal(err)
	}
}
