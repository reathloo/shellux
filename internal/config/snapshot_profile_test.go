package config

import (
	"reflect"
	"testing"
)

func TestSnapshotPresetShowsDiagnosticsWithoutPersonalDetails(t *testing.T) {
	base := Default()
	base.Style = "aurora"
	base.Background = "#123456"
	if err := base.SelectProfile("snapshot"); err != nil {
		t.Fatal(err)
	}
	original := base.Clone()
	if got := base.ActiveProfile(t.TempDir(), 70, 15); got != "snapshot" {
		t.Fatalf("manual Snapshot selected %q", got)
	}
	effective := base.EffectiveProfile(t.TempDir(), 70, 15)
	if effective.RuntimeProfile != "snapshot" || effective.Style != base.Style || effective.Animation != base.Animation || effective.Background != base.Background {
		t.Fatal("Snapshot changed the selected appearance")
	}
	for _, name := range []string{"date", "platform", "shell", "uptime", "cpu", "ram", "load-average", "network-status"} {
		if !EntryShown(effective.Visible, name) {
			t.Fatalf("diagnostic entry %s hidden", name)
		}
	}
	for _, name := range []string{"hostname", "ip", "directory", "spotify", "git-branch", "network-status.ip", "network-status.name", "cpu.top", "ram.pressure"} {
		if Shown(effective.Visible, name) {
			t.Fatalf("personal detail %s visible", name)
		}
	}
	if err := effective.Validate(); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(base, original) {
		t.Fatal("Snapshot changed saved settings")
	}
}
