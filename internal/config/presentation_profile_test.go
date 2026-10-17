package config

import (
	"reflect"
	"testing"
)

func TestPresentationPresetUsesOnlyPublicMetrics(t *testing.T) {
	base := Default()
	base.Style = "aurora"
	base.Background = "#123456"
	for _, name := range []string{"platform", "shell", "volume", "battery"} {
		base.Visible[name] = false
	}
	base.Visible["ip"] = true
	base.Visible["spotify"] = true
	base.Visible["logged-users"] = true
	base.Visible["git-branch"] = true
	if err := base.SelectProfile("presentation"); err != nil {
		t.Fatal(err)
	}
	original := base.Clone()
	if got := base.ActiveProfile(t.TempDir(), 70, 15); got != "presentation" {
		t.Fatalf("manual Presentation selected %q", got)
	}
	effective := base.EffectiveProfile(t.TempDir(), 70, 15)
	if effective.RuntimeProfile != "presentation" || effective.Style != base.Style || effective.Animation != base.Animation || effective.Background != base.Background {
		t.Fatal("Presentation changed theme or failed to activate")
	}
	allowed := map[string]bool{"time": true, "date": true, "uptime": true, "cpu": true, "ram": true, "network-status": true}
	for _, item := range menuItems {
		if got := EntryShown(effective.Visible, item.name); got != allowed[item.name] {
			t.Fatalf("presentation entry %s visible = %t", item.name, got)
		}
	}
	for _, key := range []string{"date.timezone", "date.utc", "cpu.top", "ram.pressure", "network-status.connection", "network-status.name", "network-status.interface", "network-status.ip"} {
		if Shown(effective.Visible, key) {
			t.Fatalf("sensitive detail %s visible", key)
		}
	}
	if err := effective.Validate(); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(base, original) {
		t.Fatal("Presentation changed saved configuration")
	}
}
