package config

import (
	"os"
	"testing"
)

func TestNewComponentsPreserveDefaultsAndPersist(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if MenuSlots(Default().Visible) != 11 || MenuSlots(nil) != 11 {
		t.Fatal("legacy/default header slot count changed")
	}
	for _, key := range []string{"cpu.top", "ram.pressure", "battery.bar", "battery.remaining", "network-status.connection", "network-status.name", "network-traffic.latency"} {
		if Shown(nil, key) || Shown(Default().Visible, key) {
			t.Fatalf("%s enabled by default", key)
		}
	}
	for _, target := range [][2]string{{"cpu", "top"}, {"ram", "pressure"}, {"battery", "bar"}, {"battery", "remaining"}, {"status", "connection"}, {"status", "name"}, {"status", "interface"}, {"status", "ip"}, {"status", "value"}, {"traffic", "upload"}, {"traffic", "download"}, {"traffic", "latency"}} {
		for _, enabled := range []bool{true, false} {
			if _, err := SetDisplay(target[0], target[1], enabled); err != nil {
				t.Fatal(err)
			}
			c, err := LoadDefault()
			if err != nil || Shown(c.Visible, canonicalMenuItem(target[0])+"."+target[1]) != enabled {
				t.Fatalf("%v not persisted: %v", target, err)
			}
		}
	}
	if _, err := UpdateMenu("reset", nil); err != nil {
		t.Fatal(err)
	}
	c, err := LoadDefault()
	if err != nil || Shown(c.Visible, "cpu.top") || !Shown(c.Visible, "network-status.ip") || Shown(c.Visible, "battery.bar") {
		t.Fatal("reset did not restore defaults")
	}
}

func TestBatteryThermalSlotsWithHiddenDetails(t *testing.T) {
	c := Default()
	base := MenuSlots(c.Visible)
	if err := c.ChangeDisplay("thermal", "", true); err != nil {
		t.Fatal(err)
	}
	for _, part := range DisplayParts("battery") {
		if err := c.ChangeDisplay("battery", part, false); err != nil {
			t.Fatal(err)
		}
	}
	if EntryShown(c.Visible, "battery") || MenuSlots(c.Visible) != base {
		t.Fatal("standalone thermal lost its shared slot")
	}
	if err := c.ChangeDisplay("thermal", "", false); err != nil {
		t.Fatal(err)
	}
	if MenuSlots(c.Visible) != base-1 {
		t.Fatal("empty shared slot still occupied")
	}
	if err := c.ChangeDisplay("hostname", "", true); err != nil {
		t.Fatal(err)
	}
	if err := c.ChangeDisplay("battery", "remaining", true); err == nil {
		t.Fatal("battery detail exceeded slot limit")
	}
}

func TestDateDetailsPersistAndRespectSlots(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if Shown(nil, "date.timezone") || Shown(nil, "date.utc") || !Shown(nil, "date.value") {
		t.Fatal("date component defaults changed")
	}
	if _, err := UpdateMenu("replace", []string{"platform", "date"}); err != nil {
		t.Fatal(err)
	}
	for _, part := range []string{"timezone", "utc"} {
		for _, enabled := range []bool{true, false, true} {
			if _, err := SetDisplay("date", part, enabled); err != nil {
				t.Fatal(err)
			}
			c, err := LoadDefault()
			if err != nil || Shown(c.Visible, "date."+part) != enabled || MenuSlots(c.Visible) != 11 {
				t.Fatalf("%s persistence/slots failed: %v", part, err)
			}
		}
	}
	for _, enabled := range []bool{false, true} {
		if _, err := SetDisplay("date", "", enabled); err != nil {
			t.Fatal(err)
		}
		c, err := LoadDefault()
		if err != nil || !Shown(c.Visible, "date.timezone") || !Shown(c.Visible, "date.utc") {
			t.Fatal("whole-entry toggle lost details")
		}
	}
	for _, part := range DisplayParts("date") {
		if _, err := SetDisplay("date", part, false); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := SetDisplay("ip", "", true); err != nil {
		t.Fatal(err)
	}
	if _, err := SetDisplay("date", "utc", true); err == nil {
		t.Fatal("restoring a date detail exceeded the limit")
	}
	if _, err := UpdateMenu("reset", nil); err != nil {
		t.Fatal(err)
	}
	c, err := LoadDefault()
	if err != nil || Shown(c.Visible, "date.timezone") || Shown(c.Visible, "date.utc") || !Shown(c.Visible, "date.value") {
		t.Fatal("reset lost date defaults")
	}
}

func TestMenuSelectionPersistsAndPreservesAppearance(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if _, err := SetStyle("neon"); err != nil {
		t.Fatal(err)
	}
	for _, change := range []struct {
		action string
		names  []string
	}{
		{"set", []string{"cpu", "ram", "battery"}},
		{"replace", []string{"battery", "temperature"}},
		{"set", []string{"cpu", "temperature", "directory"}},
	} {
		if _, err := UpdateMenu(change.action, change.names); err != nil {
			t.Fatal(err)
		}
	}
	settings, err := LoadDefault()
	if err != nil {
		t.Fatal(err)
	}
	if settings.Style != "neon" {
		t.Fatal("menu changed style")
	}
	for _, name := range MenuItems() {
		want := name == "cpu" || name == "temperature" || name == "directory"
		if settings.Visible[name] != want {
			t.Fatalf("%s visibility = %v, want %v", name, settings.Visible[name], want)
		}
	}
	if _, err := UpdateMenu("set", nil); err != nil {
		t.Fatal(err)
	}
	settings, err = LoadDefault()
	if err != nil {
		t.Fatal(err)
	}
	for name, shown := range settings.Visible {
		if shown {
			t.Fatalf("empty selection still shows %s", name)
		}
	}
	if _, err := UpdateMenu("reset", nil); err != nil {
		t.Fatal(err)
	}
	settings, err = LoadDefault()
	if err != nil {
		t.Fatal(err)
	}
	for name, shown := range Default().Visible {
		if settings.Visible[name] != shown {
			t.Fatalf("reset did not restore %s", name)
		}
	}
}

func TestInvalidMenuDoesNotChangeSavedConfig(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	path, err := UpdateMenu("set", []string{"cpu", "ram"})
	if err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, change := range []struct {
		action string
		names  []string
	}{
		{"set", []string{"cpu", "unknown"}}, {"set", []string{"cpu", "cpu"}},
		{"replace", []string{"battery", "temperature"}}, {"replace", []string{"cpu", "ram"}},
		{"add", []string{"cpu"}}, {"remove", []string{"cpu"}}, {"reset", []string{"cpu"}}, {"unknown", nil},
	} {
		if _, err := UpdateMenu(change.action, change.names); err == nil {
			t.Fatalf("accepted %+v", change)
		}
		after, err := os.ReadFile(path)
		if err != nil || string(before) != string(after) {
			t.Fatal("invalid command changed config")
		}
	}
}

func TestDisplayPartsPersistAcrossWholeEntryToggle(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	path, err := SetDisplay("cpu", "bar", false)
	if err != nil {
		t.Fatal(err)
	}
	for _, enabled := range []bool{false, true} {
		if _, err := SetDisplay("cpu", "", enabled); err != nil {
			t.Fatal(err)
		}
		settings, err := LoadDefault()
		if err != nil {
			t.Fatal(err)
		}
		if settings.Visible["cpu"] != enabled || Shown(settings.Visible, "cpu.bar") || !Shown(settings.Visible, "cpu.percent") {
			t.Fatalf("lost component settings: %v", settings.Visible)
		}
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, target := range [][2]string{{"cpu", "bogus"}, {"battery", "pressure"}, {"unknown", ""}} {
		if _, err := SetDisplay(target[0], target[1], false); err == nil {
			t.Fatalf("accepted %v", target)
		}
	}
	after, err := os.ReadFile(path)
	if err != nil || string(before) != string(after) {
		t.Fatal("invalid display setting changed config")
	}
	if _, err := UpdateMenu("reset", nil); err != nil {
		t.Fatal(err)
	}
	settings, err := LoadDefault()
	if err != nil || !Shown(settings.Visible, "cpu.bar") {
		t.Fatalf("reset did not restore parts: %+v, %v", settings, err)
	}
}

func TestAlternativeEntriesAreHiddenInLegacyConfig(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	path, err := SetStyle("neon")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"visible":{"cpu":true}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	settings, err := LoadDefault()
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"hostname", "cores", "ip", "date"} {
		if settings.Visible[name] || Shown(nil, name) {
			t.Fatalf("new entry %s became visible by default", name)
		}
	}
}

func TestAlternativesRespectEntryLimitWithoutChangingConfig(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	path, err := UpdateMenu("reset", nil)
	if err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := SetDisplay("hostname", "", true); err == nil {
		t.Fatal("accepted twelfth entry")
	}
	after, err := os.ReadFile(path)
	if err != nil || string(before) != string(after) {
		t.Fatal("limit failure changed config")
	}
	if _, err := UpdateMenu("replace", []string{"battery", "hostname"}); err != nil {
		t.Fatal(err)
	}
	settings, err := LoadDefault()
	if err != nil || !settings.Visible["hostname"] || settings.Visible["battery"] {
		t.Fatalf("replacement failed: %+v %v", settings, err)
	}
	// Disabling every component frees an actual visible entry slot.
	for _, part := range DisplayParts("cpu") {
		if _, err := SetDisplay("cpu", part, false); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := SetDisplay("date", "", true); err != nil {
		t.Fatal(err)
	}
	if _, err := SetDisplay("cpu", "bar", true); err == nil {
		t.Fatal("restoring CPU exceeded visible entry limit")
	}
}

func TestMenuAliasesUseExistingSettings(t *testing.T) {
	for _, entry := range []struct{ alias, key string }{
		{"path", "directory"}, {"traffic", "network-traffic"}, {"status", "network-status"}, {"thermal", "temperature"},
	} {
		t.Run(entry.alias, func(t *testing.T) {
			t.Setenv("XDG_CONFIG_HOME", t.TempDir())
			if _, err := UpdateMenu("set", []string{entry.alias}); err != nil {
				t.Fatal(err)
			}
			for _, enabled := range []bool{false, true} {
				if _, err := SetDisplay(entry.alias, "", enabled); err != nil {
					t.Fatal(err)
				}
				settings, err := LoadDefault()
				if err != nil || settings.Visible[entry.key] != enabled {
					t.Fatalf("alias failed: %+v %v", settings, err)
				}
				if _, exists := settings.Visible[entry.alias]; exists {
					t.Fatal("alias created a separate config entry")
				}
			}
			if _, err := UpdateMenu("replace", []string{entry.alias, "ip"}); err != nil {
				t.Fatal(err)
			}
			if _, err := UpdateMenu("replace", []string{"ip", entry.alias}); err != nil {
				t.Fatal(err)
			}
			if _, err := UpdateMenu("set", []string{"cpu", entry.alias}); err != nil {
				t.Fatal(err)
			}
			settings, err := LoadDefault()
			if err != nil || !settings.Visible[entry.key] {
				t.Fatalf("set with alias failed: %+v %v", settings, err)
			}
			if _, err := UpdateMenu("set", []string{entry.alias, entry.key}); err == nil {
				t.Fatal("accepted duplicate alias")
			}
			if DisplayMenuItem(entry.key) != entry.alias {
				t.Fatal("list does not use public alias")
			}
		})
	}
}

func TestCoresPartsPersistAndRespectEntryLimit(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if _, err := UpdateMenu("replace", []string{"cpu", "cores"}); err != nil {
		t.Fatal(err)
	}
	for _, part := range DisplayParts("cores") {
		for _, enabled := range []bool{false, true} {
			if _, err := SetDisplay("cores", part, enabled); err != nil {
				t.Fatal(err)
			}
			settings, err := LoadDefault()
			if err != nil || Shown(settings.Visible, "cores."+part) != enabled {
				t.Fatalf("cores.%s did not persist: %v", part, err)
			}
		}
	}
	for _, part := range DisplayParts("cores") {
		if _, err := SetDisplay("cores", part, false); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := SetDisplay("ip", "", true); err != nil {
		t.Fatal(err)
	}
	if _, err := SetDisplay("cores", "bar", true); err == nil {
		t.Fatal("restoring cores exceeded entry limit")
	}
}

func TestLegacyPlaybackMigratesToSeparateEntries(t *testing.T) {
	for _, test := range []struct {
		name, data        string
		spotify, progress bool
	}{
		{"legacy on", `{"visible":{"playback":true}}`, true, true},
		{"legacy off", `{"visible":{"playback":false}}`, false, false},
		{"explicit spotify", `{"visible":{"playback":false,"spotify":true}}`, true, false},
		{"explicit progress", `{"visible":{"playback":true,"progress":false}}`, true, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("XDG_CONFIG_HOME", t.TempDir())
			path, err := UpdateMenu("reset", nil)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte(test.data), 0o600); err != nil {
				t.Fatal(err)
			}
			settings, err := LoadDefault()
			if err != nil {
				t.Fatal(err)
			}
			if settings.Visible["spotify"] != test.spotify || settings.Visible["progress"] != test.progress {
				t.Fatalf("migration = %v", settings.Visible)
			}
			if _, exists := settings.Visible["playback"]; exists {
				t.Fatal("legacy setting was retained")
			}
			if _, err := SetDisplay("spotify", "", !test.spotify); err != nil {
				t.Fatal(err)
			}
			settings, err = LoadDefault()
			if err != nil || settings.Visible["progress"] != test.progress {
				t.Fatalf("spotify changed progress: %+v %v", settings, err)
			}
			if _, err := SetDisplay("playback", "", false); err == nil {
				t.Fatal("removed playback command accepted")
			}
		})
	}
}
