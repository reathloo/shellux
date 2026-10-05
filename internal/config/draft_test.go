package config

import (
	"errors"
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
