package systeminfo

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCodingInfoUsesEnabledLocalTools(t *testing.T) {
	// Busy CI runners can delay child processes; this test checks selection and
	// parsing, while production probes retain their short timeout.
	previousTimeout := codingCommandTimeout
	codingCommandTimeout = 10 * time.Second
	t.Cleanup(func() { codingCommandTimeout = previousTimeout })
	directory := t.TempDir()
	bin := filepath.Join(directory, "bin")
	if err := os.Mkdir(bin, 0700); err != nil {
		t.Fatal(err)
	}
	log := filepath.Join(directory, "calls")
	for name, output := range map[string]string{
		"go": "go version go1.25.1 darwin/arm64", "node": "v22.5.0",
		"python3": "Python 3.13.2", "java": `openjdk version "21.0.2"`,
		"docker": "desktop-linux", "kubectl": "dev-cluster",
	} {
		script := "#!/bin/sh\nprintf '%s\\n' " + name + " >> " + log + "\nprintf '%s\\n' '" + output + "'\n"
		if err := os.WriteFile(filepath.Join(bin, name), []byte(script), 0700); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	visible := map[string]bool{"runtimes": false, "dev-context": false}
	CodingInfo(visible, directory)
	if _, err := os.Stat(log); !os.IsNotExist(err) {
		t.Fatal("hidden coding tools were executed")
	}
	visible["runtimes"], visible["dev-context"] = true, true
	visible["runtimes.java"] = false
	first := CodingInfo(visible, directory)
	if first.Go != "1.25.1" || first.Node != "22.5.0" || first.Python != "3.13.2" || first.Java != "" || first.Docker != "desktop-linux" || first.Kubernetes != "dev-cluster" {
		codingCommands.Lock()
		for key, sample := range codingCommands.values {
			if strings.HasPrefix(key, directory+"\x00") && sample.err != nil {
				t.Logf("probe %q: %v", strings.TrimPrefix(key, directory+"\x00"), sample.err)
			}
		}
		codingCommands.Unlock()
		t.Fatalf("coding tools = %+v", first)
	}
	CodingInfo(visible, directory)
	data, err := os.ReadFile(log)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(data), "\n") != 5 || strings.Contains(string(data), "java") {
		t.Fatalf("disabled or cached tools ran: %q", data)
	}
	visible["runtimes.java"] = true
	if got := CodingInfo(visible, directory).Java; got != "21.0.2" {
		t.Fatalf("Java version = %q", got)
	}
}

func TestGitCodingInfoUsesLocalUpstream(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git unavailable")
	}
	directory := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = directory
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %s: %v", args, output, err)
		}
	}
	run("init", "-q")
	run("checkout", "-qb", "main")
	run("config", "user.name", "Shellux Test")
	run("config", "user.email", "shellux@example.invalid")
	if err := os.WriteFile(filepath.Join(directory, "file"), []byte("first"), 0600); err != nil {
		t.Fatal(err)
	}
	run("add", "file")
	run("commit", "-qm", "first")
	run("branch", "upstream")
	if err := os.WriteFile(filepath.Join(directory, "file"), []byte("second"), 0600); err != nil {
		t.Fatal(err)
	}
	run("commit", "-qam", "second")
	run("branch", "--set-upstream-to=upstream")
	visible := map[string]bool{"git-branch": true, "git-sync": true}
	got := CodingInfo(visible, directory)
	if got.Branch != "main" || got.Repository != filepath.Base(directory) || !got.SyncAvailable || got.Ahead != 1 || got.Behind != 0 {
		t.Fatalf("Git context = %+v", got)
	}
}

func TestLastCommandEnvironmentIsOptional(t *testing.T) {
	t.Setenv("SHELLUX_LAST_COMMAND_SECONDS", "2")
	t.Setenv("SHELLUX_LAST_COMMAND_STATUS", "7")
	visible := map[string]bool{"last-command": true}
	got := CodingInfo(visible, t.TempDir())
	if !got.CommandAvailable || got.CommandSeconds != 2 || got.CommandExit != 7 {
		t.Fatalf("last command = %+v", got)
	}
	t.Setenv("SHELLUX_LAST_COMMAND_STATUS", "invalid")
	if CodingInfo(visible, t.TempDir()).CommandAvailable {
		t.Fatal("invalid shell data was accepted")
	}
}
