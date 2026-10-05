package config

import "testing"

func TestLanguageDraftPersistence(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	editor, err := OpenEditor()
	if err != nil {
		t.Fatal(err)
	}
	if editor.Config.Language != "auto" {
		t.Fatal("new config must use system language")
	}
	draft := editor.Config.Clone()
	if err := draft.ChangeAppearance("language", "de"); err != nil {
		t.Fatal(err)
	}
	unchanged, err := LoadDefault()
	if err != nil || unchanged.Language != "auto" {
		t.Fatal("draft leaked before save")
	}
	if err := draft.ChangeAppearance("language", "xx"); err == nil || draft.Language != "de" {
		t.Fatal("invalid language changed draft")
	}
	if err := editor.Save(draft); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadDefault()
	if err != nil || loaded.Language != "de" {
		t.Fatalf("save: %+v %v", loaded, err)
	}
	legacy, err := decodeConfig([]byte(`{"refresh_interval":"1s"}`))
	if err != nil || legacy.Language != "auto" {
		t.Fatalf("legacy: %+v %v", legacy, err)
	}
	loaded.ResetMenu()
	if loaded.Language != "de" {
		t.Fatal("menu reset changed language")
	}
}
