package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestRemoteProfileDetectionAndLayout(t *testing.T) {
	for _, name := range []string{"SSH_CONNECTION", "SSH_CLIENT", "SSH_TTY"} {
		t.Setenv(name, "")
	}
	t.Setenv("SSH_AUTH_SOCK", "/tmp/agent.sock")
	project := t.TempDir()
	if err := os.WriteFile(filepath.Join(project, "go.mod"), []byte("module example.test/project\n"), 0600); err != nil {
		t.Fatal(err)
	}
	c := Default()
	c.Animation = "shelluxdefault"
	c.Style = "aurora"
	c.Background = "#123456"
	if err := c.SetProfileSelection("auto"); err != nil {
		t.Fatal(err)
	}
	if got := c.ActiveProfile(t.TempDir(), 140, 40); got != "" {
		t.Fatalf("SSH agent alone selected %q", got)
	}
	if got := c.ActiveProfile(project, 140, 40); got != "coding" {
		t.Fatalf("local project selected %q", got)
	}
	for _, variable := range []string{"SSH_CONNECTION", "SSH_CLIENT", "SSH_TTY"} {
		t.Setenv(variable, "remote")
		if got := c.ActiveProfile(project, 140, 40); got != "remote" {
			t.Fatalf("%s selected %q", variable, got)
		}
		t.Setenv(variable, "")
	}
	t.Setenv("SSH_CONNECTION", "client 12345 server 22")
	if got := c.ActiveProfile(project, 90, 40); got != "minimal" {
		t.Fatalf("narrow SSH terminal selected %q", got)
	}
	before := c.Clone()
	effective := c.EffectiveProfile(project, 140, 40)
	if effective.RuntimeProfile != "remote" || effective.Animation != c.Animation || effective.Style != c.Style || effective.Background != c.Background {
		t.Fatal("Remote did not preserve the selected theme")
	}
	for _, key := range []string{"remote-identity.user", "remote-identity.host", "ssh-link.client", "ssh-link.server", "remote-terminal.tty", "remote-terminal.term", "uptime", "directory", "network-status.value", "network-status.interface", "cpu.percent", "ram.percent", "time"} {
		if !Shown(effective.Visible, key) {
			t.Fatalf("Remote hid %s", key)
		}
	}
	for _, key := range []string{"hostname", "shell", "ip", "spotify", "progress", "network-traffic", "cpu.bar", "ram.bar", "runtimes", "git-branch"} {
		if Shown(effective.Visible, key) {
			t.Fatalf("Remote showed %s", key)
		}
	}
	if !reflect.DeepEqual(effective.MenuLayout.Groups[:4], []string{"REMOTE", "RESOURCES", "SESSION", "NETWORK"}) {
		t.Fatalf("Remote group order = %v", effective.MenuLayout.Groups)
	}
	if err := effective.Validate(); err != nil {
		t.Fatalf("Remote layout invalid: %v", err)
	}
	if !reflect.DeepEqual(c, before) {
		t.Fatal("Remote changed saved settings")
	}
	if err := c.SelectProfile("remote"); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SSH_CONNECTION", "")
	if got := c.ActiveProfile(project, 140, 40); got != "remote" {
		t.Fatalf("manual Remote selected %q", got)
	}
}

func TestRemoteMenuItemsHaveIndependentDetails(t *testing.T) {
	for name, parts := range map[string][]string{
		"remote-identity": {"user", "host"},
		"ssh-link":        {"client", "server"},
		"remote-terminal": {"tty", "term"},
	} {
		if got := DisplayParts(name); !reflect.DeepEqual(got, parts) {
			t.Fatalf("%s parts = %v", name, got)
		}
		c := Default()
		if err := c.ChangeDisplay("platform", "", false); err != nil {
			t.Fatal(err)
		}
		if err := c.ChangeDisplay(name, "", true); err != nil {
			t.Fatal(err)
		}
		if err := c.ChangeDisplay(name, parts[0], false); err != nil {
			t.Fatal(err)
		}
		if Shown(c.Visible, name+"."+parts[0]) || !Shown(c.Visible, name+"."+parts[1]) || !EntryShown(c.Visible, name) {
			t.Fatalf("%s detail switches are coupled", name)
		}
	}
}
