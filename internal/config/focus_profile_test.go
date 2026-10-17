package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFocusTimerRestoresPreviousAutomaticLayoutAndTheme(t *testing.T) {
	project := t.TempDir()
	if err := os.WriteFile(filepath.Join(project, "go.mod"), []byte("module example.test/focus\n"), 0600); err != nil {
		t.Fatal(err)
	}
	c := Default()
	c.Style, c.Background = "aurora", "#123456"
	if err := c.SetProfileSelection("auto"); err != nil {
		t.Fatal(err)
	}
	if err := c.SetFocusDuration(30); err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	c.ActivateFocus(start)
	if got := c.ActiveProfile(project, 140, 40); got != "focus" {
		t.Fatalf("active focus = %q", got)
	}
	effective := c.EffectiveProfile(project, 140, 40)
	if effective.RuntimeProfile != "focus" || effective.Background != "#000000" || effective.Style != "aurora" {
		t.Fatal("Focus did not apply its temporary appearance")
	}
	for _, key := range []string{"cpu.percent", "ram.percent", "spotify", "progress"} {
		if !Shown(effective.Visible, key) {
			t.Fatalf("Focus hid %s", key)
		}
	}
	for _, key := range []string{"directory", "git-branch", "network-status", "cpu.bar", "ram.bar"} {
		if Shown(effective.Visible, key) {
			t.Fatalf("Focus showed %s", key)
		}
	}
	if remaining := c.FocusRemaining(start.Add(10 * time.Minute)); remaining != 20*time.Minute {
		t.Fatalf("remaining = %s", remaining)
	}
	c.FocusUntil = time.Now().Add(-time.Second).Format(time.RFC3339Nano)
	if got := c.ActiveProfile(project, 140, 40); got != "coding" {
		t.Fatalf("expired focus resolved %q", got)
	}
	restored := c.EffectiveProfile(project, 140, 40)
	if restored.RuntimeProfile != "coding" || restored.Background != "#123456" || restored.Style != "aurora" {
		t.Fatal("expired focus did not restore saved appearance")
	}
	data, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := decodeConfig(data)
	if err != nil || loaded.ProfileSelection != "auto" || loaded.SelectedProfile != "" || loaded.FocusUntil != "" {
		t.Fatalf("expired focus load = %+v, %v", loaded, err)
	}
}

func TestFocusDurationValidation(t *testing.T) {
	c := Default()
	for _, minutes := range []int{0, 241} {
		if err := c.SetFocusDuration(minutes); err == nil || c.SelectedProfile != "" {
			t.Fatalf("invalid duration %d changed selection", minutes)
		}
	}
}
