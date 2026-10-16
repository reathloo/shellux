package config

import (
	"errors"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestDraftOwnsMapsAndUsesSharedRules(t *testing.T) {
	original := Default()
	original.Themes = map[string]SavedTheme{"my-look": {Animation: "orb", Style: "neon", Background: "#123456"}}
	draft := original.Clone()
	if err := draft.ChangeDisplay("cpu", "bar", false); err != nil {
		t.Fatal(err)
	}
	if _, err := draft.RemoveTheme("my-look"); err != nil {
		t.Fatal(err)
	}
	if !Shown(original.Visible, "cpu.bar") || len(original.Themes) != 1 {
		t.Fatal("draft mutated baseline")
	}
	draft.ResetMenu()
	if !reflect.DeepEqual(draft.Visible, Default().Visible) || draft.Background != original.Background {
		t.Fatal("reset changed appearance or retained details")
	}
	for _, e := range MenuEntries() {
		if e.Key == "cores" && e.Group != "RESOURCES" {
			t.Fatal("cores group")
		}
	}
}

func TestMenuLayoutMovesCloneAndReset(t *testing.T) {
	c := Default()
	if !c.MoveMenuGroup("RESOURCES", -1) || !c.MoveMenuGroup("RESOURCES", -1) {
		t.Fatal("resources group did not move to the top")
	}
	if got := c.MenuLayout.Groups[0]; got != "RESOURCES" {
		t.Fatalf("first group = %q", got)
	}
	if !c.MoveMenuItem("cpu", 1) || c.MenuLayout.Items["RESOURCES"][1] != "cpu" {
		t.Fatalf("CPU did not move inside resources: %v", c.MenuLayout.Items["RESOURCES"])
	}
	if !c.MoveDisplayPart("cpu", "sparkline", 1) || c.MenuLayout.Components["cpu"][1] != "sparkline" {
		t.Fatalf("sparkline did not move: %v", c.MenuLayout.Components["cpu"])
	}
	clone := c.Clone()
	if !clone.MoveDisplayPart("cpu", "sparkline", 1) || reflect.DeepEqual(clone.MenuLayout, c.MenuLayout) {
		t.Fatal("layout clone shares component slices")
	}
	if c.MoveMenuGroup("HEADER", -1) {
		t.Fatal("header group moved away from its fixed title")
	}
	c.ResetMenu()
	if !reflect.DeepEqual(c.MenuLayout, Default().MenuLayout) || !reflect.DeepEqual(c.Visible, Default().Visible) {
		t.Fatal("reset did not restore visibility and ordering")
	}
}

func TestLegacyConfigGetsDefaultMenuLayout(t *testing.T) {
	c, err := decodeConfig([]byte(`{"animation":"shelluxdefault","style":"shelluxdefault","refresh_interval":"1s"}`))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(c.MenuLayout, Default().MenuLayout) {
		t.Fatalf("legacy layout = %#v", c.MenuLayout)
	}
	c.MenuLayout.Components["cpu"] = []string{"bar", "bar", "value", "percent", "top"}
	if err := c.Validate(); err == nil {
		t.Fatal("invalid component permutation accepted")
	}
}

func TestMenuLayoutPersists(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	c := Default()
	if !c.MoveMenuGroup("RESOURCES", -1) || !c.MoveMenuItem("cpu", 1) || !c.MoveDisplayPart("ram", "percent", -1) {
		t.Fatal("could not prepare custom layout")
	}
	if _, err := writeDefault(c); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadDefault()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(loaded.MenuLayout, c.MenuLayout) {
		t.Fatalf("layout changed while saving: %#v != %#v", loaded.MenuLayout, c.MenuLayout)
	}
}

func TestMenuPresetsSaveApplyDeleteAndStayIndependent(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	c := Default()
	c.Style = "neon"
	c.RefreshInterval = "500ms"
	if !c.MoveMenuGroup("RESOURCES", -1) || !c.MoveMenuItem("cpu", 1) || !c.MoveDisplayPart("cpu", "sparkline", 1) {
		t.Fatal("could not prepare menu layout")
	}
	if err := c.ChangeDisplay("platform", "", false); err != nil {
		t.Fatal(err)
	}
	if err := c.ChangeDisplay("cpu", "sparkline", true); err != nil {
		t.Fatal(err)
	}
	wantVisible := maps.Clone(c.Visible)
	wantLayout := c.Clone().MenuLayout
	if name, err := c.StoreMenuPreset("  Meine-Vorlage  "); err != nil || name != "meine-vorlage" {
		t.Fatalf("store = %q, %v", name, err)
	}
	clone := c.Clone()
	clone.MenuPresets["meine-vorlage"].Visible["cpu"] = false
	if !c.MenuPresets["meine-vorlage"].Visible["cpu"] {
		t.Fatal("preset visibility shares the cloned map")
	}
	clonedPreset := clone.MenuPresets["meine-vorlage"]
	clonedPreset.Layout.Groups[0] = "SESSION"
	if c.MenuPresets["meine-vorlage"].Layout.Groups[0] != "SYSTEM" {
		t.Fatal("preset order shares the cloned slice")
	}
	c.ResetMenu()
	if len(c.MenuPresets) != 1 || c.Style != "neon" {
		t.Fatal("reset deleted a saved preset or changed appearance")
	}
	if err := c.ApplyMenuPreset("meine-vorlage"); err != nil {
		t.Fatal(err)
	}
	if !maps.Equal(c.Visible, wantVisible) || !reflect.DeepEqual(c.MenuLayout, wantLayout) || c.Style != "neon" || c.RefreshInterval != "500ms" {
		t.Fatal("applying the preset changed an unrelated setting or lost menu choices")
	}
	if _, err := writeDefault(c); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadDefault()
	if err != nil || !loaded.MenuPresetMatches("meine-vorlage") {
		t.Fatalf("preset not persisted: %v", err)
	}
	if err := loaded.ApplyMenuPreset("default"); err != nil || !loaded.MenuPresetMatches("default") || len(loaded.MenuPresets) != 1 {
		t.Fatal("default layout removed saved presets")
	}
	if err := loaded.RemoveMenuPreset("meine-vorlage"); err != nil || len(loaded.MenuPresets) != 0 || loaded.Style != "neon" {
		t.Fatal("delete changed appearance or retained the preset")
	}
	if _, err := loaded.StoreMenuPreset("default"); err == nil {
		t.Fatal("reserved preset name accepted")
	}
	if err := loaded.RemoveMenuPreset("default"); err == nil {
		t.Fatal("built-in default preset deleted")
	}
}

func TestSlotsAndTransactionalComponentChanges(t *testing.T) {
	c := Default()
	base := MenuSlots(c.Visible)
	for _, name := range []string{"thermal", "spotify", "progress"} {
		if err := c.ChangeDisplay(name, "", true); err != nil {
			t.Fatal(err)
		}
	}
	if got := MenuSlots(c.Visible); got != base {
		t.Fatalf("shared entries added slots: %d != %d", got, base)
	}
	for _, name := range []string{"cpu", "ram", "cores", "volume"} {
		if !c.Visible[name] {
			continue
		}
		before := MenuSlots(c.Visible)
		for _, part := range DisplayParts(name) {
			if err := c.ChangeDisplay(name, part, false); err != nil {
				t.Fatal(err)
			}
		}
		if got := MenuSlots(c.Visible); got != before-1 {
			t.Fatalf("%s without components occupies a slot", name)
		}
	}
	c.ResetMenu()
	for _, e := range MenuEntries() {
		if MenuSlots(c.Visible) == 11 {
			break
		}
		if !c.Visible[e.Key] {
			_ = c.ChangeDisplay(e.Key, "", true)
		}
	}
	for _, e := range MenuEntries() {
		if c.Visible[e.Key] || e.Key == "time" || e.Key == "temperature" {
			continue
		}
		before := c.Clone()
		if err := c.ChangeDisplay(e.Key, "", true); err == nil {
			t.Fatalf("capacity accepted %s", e.Key)
		}
		if !reflect.DeepEqual(c, before) {
			t.Fatal("rejected switch mutated draft")
		}
	}
}

func TestEditorSaveCancelAndConflict(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	session, err := OpenEditor()
	if err != nil {
		t.Fatal(err)
	}
	draft := session.Config.Clone()
	if err := draft.ChangeAppearance("interval", "500ms"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(session.path); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("editing created a file")
	}
	if err := session.Save(draft); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadDefault()
	if err != nil || loaded.RefreshInterval != "500ms" {
		t.Fatalf("saved config: %v %v", loaded, err)
	}
	session, err = OpenEditor()
	if err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(session.path)
	if err != nil {
		t.Fatal(err)
	}
	draft = session.Config.Clone()
	_ = draft.ChangeDisplay("cpu", "", false)
	// Discarding a draft performs no write.
	after, _ := os.ReadFile(session.path)
	if !reflect.DeepEqual(before, after) {
		t.Fatal("cancel changed file")
	}
	// Even invalid external content and whitespace-only rewrites count as conflicts.
	external := append(append([]byte(nil), before...), ' ')
	if err := os.WriteFile(session.path, external, 0600); err != nil {
		t.Fatal(err)
	}
	if err := session.Save(draft); !errors.Is(err, ErrChanged) {
		t.Fatalf("conflict = %v", err)
	}
	after, _ = os.ReadFile(session.path)
	if !reflect.DeepEqual(external, after) {
		t.Fatal("overwrote external changes")
	}
}

func TestEditorCreationDeletionInvalidAndWriteFailure(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	session, err := OpenEditor()
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := SetBackground("navy"); err != nil {
		t.Fatal(err)
	}
	if err := session.Save(session.Config); !errors.Is(err, ErrChanged) {
		t.Fatalf("external creation = %v", err)
	}
	session, err = OpenEditor()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(session.path); err != nil {
		t.Fatal(err)
	}
	if err := session.Save(session.Config); !errors.Is(err, ErrChanged) {
		t.Fatalf("external deletion = %v", err)
	}
	session, err = OpenEditor()
	if err != nil {
		t.Fatal(err)
	}
	invalid := session.Config.Clone()
	invalid.RefreshInterval = "0s"
	if err := session.Save(invalid); err == nil {
		t.Fatal("saved invalid duration")
	}
	if err := os.Remove(filepath.Dir(session.path)); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Dir(session.path), []byte("block directory"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := session.Save(session.Config); err == nil {
		t.Fatal("write failure accepted")
	}
}

func TestDraftThemesAndAppearance(t *testing.T) {
	c := Default()
	for _, v := range []struct{ field, value string }{{"animation", "orb"}, {"style", "neon"}, {"background", "#123456"}, {"interval", "500ms"}} {
		if err := c.ChangeAppearance(v.field, v.value); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := c.StoreTheme("my-look"); err != nil {
		t.Fatal(err)
	}
	if c.BackgroundName() != "my-look" {
		t.Fatal("custom background name missing")
	}
	if _, err := c.ApplyTheme("purplemonster"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.ApplyTheme("my-look"); err != nil {
		t.Fatal(err)
	}
	if c.Animation != "orb" || c.Style != "neon" || c.Background != "#123456" {
		t.Fatal("incomplete theme")
	}
	if _, err := c.RemoveTheme("purplemonster"); err == nil {
		t.Fatal("deleted built-in")
	}
	if _, err := c.StoreTheme("default"); err == nil {
		t.Fatal("overwrote built-in")
	}
	for _, v := range []struct{ field, value string }{{"animation", "bad"}, {"style", "bad"}, {"background", "#xyz"}, {"interval", "-1s"}} {
		before := c.Clone()
		if err := c.ChangeAppearance(v.field, v.value); err == nil {
			t.Fatal("invalid appearance accepted")
		}
		if !reflect.DeepEqual(c, before) {
			t.Fatal("invalid input changed draft")
		}
	}
	for range 20 {
		if _, err := c.ApplyTheme("random"); err != nil {
			t.Fatal(err)
		}
		if err := c.Validate(); err != nil {
			t.Fatal(err)
		}
	}
}
