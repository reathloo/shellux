package config

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestTrafficMigration(t *testing.T) {
	for _, tc := range []struct {
		name, saved                      string
		entry, upload, download, latency bool
	}{
		{"default", `{"upload":false,"download":false,"latency":false}`, true, true, true, false},
		{"with ping", `{"network-traffic":true,"latency":true}`, true, true, true, true},
		{"only ping", `{"network-traffic":false,"latency":true}`, true, false, false, true},
		{"only upload", `{"network-traffic":false,"upload":true,"download":false}`, true, true, false, false},
		{"only download", `{"network-traffic":false,"download":true}`, true, false, true, false},
		{"all hidden", `{"network-traffic":false,"upload":false,"download":false,"latency":false}`, false, true, true, false},
		{"no duplicates", `{"network-traffic":true,"upload":true,"download":true}`, true, true, true, false},
		{"new preferences win", `{"network-traffic":false,"latency":true,"network-traffic.upload":false,"network-traffic.latency":false}`, false, false, false, false},
		{"current format", `{"network-traffic":true,"network-traffic.upload":false,"network-traffic.download":false,"network-traffic.latency":true}`, true, false, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Disable other entries so every historical selection is valid.
			c := Default()
			for key := range c.Visible {
				c.Visible[key] = false
			}
			var saved map[string]bool
			if err := json.Unmarshal([]byte(tc.saved), &saved); err != nil {
				t.Fatal(err)
			}
			// Omitted Traffic uses the old default (both transfers enabled).
			c.Visible["network-traffic"] = true
			for key, value := range saved {
				c.Visible[key] = value
			}
			data, err := json.Marshal(c)
			if err != nil {
				t.Fatal(err)
			}
			loaded, err := decodeConfig(data)
			if err != nil {
				t.Fatal(err)
			}
			for key, want := range map[string]bool{"network-traffic": tc.entry, "network-traffic.upload": tc.upload, "network-traffic.download": tc.download, "network-traffic.latency": tc.latency} {
				if Shown(loaded.Visible, key) != want {
					t.Fatalf("%s = %v, want %v", key, Shown(loaded.Visible, key), want)
				}
			}
			for _, key := range []string{"upload", "download", "latency"} {
				if _, exists := loaded.Visible[key]; exists {
					t.Fatalf("legacy %s remains", key)
				}
			}
			wantSlots := 0
			if tc.entry && (tc.upload || tc.download || tc.latency) {
				wantSlots = 1
			}
			if MenuSlots(loaded.Visible) != wantSlots {
				t.Fatal("merged Traffic slot count")
			}
		})
	}
}

func TestTrafficMigrationOnlyPersistsOnSave(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	path, err := DefaultPath()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	original := []byte(`{"visible":{"network-traffic":false,"latency":true}}`)
	if err := os.WriteFile(path, original, 0600); err != nil {
		t.Fatal(err)
	}
	session, err := OpenEditor()
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(path); !bytes.Equal(got, original) {
		t.Fatal("loading rewrote config")
	}
	if err := session.Save(session.Config); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var stored Config
	if err := json.Unmarshal(data, &stored); err != nil {
		t.Fatal(err)
	}
	if !stored.Visible["network-traffic.latency"] || stored.Visible["network-traffic.upload"] || stored.Visible["network-traffic.download"] {
		t.Fatal("migration not persisted")
	}
	if _, exists := stored.Visible["latency"]; exists {
		t.Fatal("old latency item persisted")
	}
}

func TestTrafficComponentsRespectSlotsAndPreserveChoices(t *testing.T) {
	c := Default()
	for _, name := range []string{"upload", "download", "latency"} {
		if err := c.ChangeDisplay(name, "", true); err == nil {
			t.Fatalf("%s still a standalone item", name)
		}
	}
	if err := c.ChangeDisplay("traffic", "latency", true); err != nil {
		t.Fatal(err)
	}
	if MenuSlots(c.Visible) != 11 {
		t.Fatal("latency consumed another slot")
	}
	if err := c.ChangeDisplay("traffic", "", false); err != nil {
		t.Fatal(err)
	}
	if !Shown(c.Visible, "network-traffic.latency") {
		t.Fatal("hiding Traffic lost details")
	}
	if err := c.ChangeDisplay("traffic", "", true); err != nil {
		t.Fatal(err)
	}
	for _, part := range DisplayParts("traffic") {
		if err := c.ChangeDisplay("traffic", part, false); err != nil {
			t.Fatal(err)
		}
	}
	if EntryShown(c.Visible, "traffic") || MenuSlots(c.Visible) != 10 {
		t.Fatal("empty Traffic still occupies a slot")
	}
	if err := c.ChangeDisplay("hostname", "", true); err != nil {
		t.Fatal(err)
	}
	if err := c.ChangeDisplay("traffic", "upload", true); err == nil {
		t.Fatal("detail bypassed slot limit")
	}
	if Shown(c.Visible, "network-traffic.upload") {
		t.Fatal("failed change modified draft")
	}
	if err := c.ChangeDisplay("hostname", "", false); err != nil {
		t.Fatal(err)
	}
	if err := c.ChangeDisplay("traffic", "upload", true); err != nil {
		t.Fatal(err)
	}
	c.ResetMenu()
	if !Shown(c.Visible, "network-traffic.upload") || !Shown(c.Visible, "network-traffic.download") || Shown(c.Visible, "network-traffic.latency") {
		t.Fatal("reset lost Traffic defaults")
	}
}
