package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/reathloo/shellux/internal/config"
	"github.com/reathloo/shellux/internal/themepack"
)

func TestAppearanceCommandRequiresSuccessfulInstall(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("SHELLUX_THEME_CATALOG_URL", "https://127.0.0.1:1/catalog.json")
	_, path, err := saveAppearance("style", "neon")
	if err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := saveAppearance("theme", "heart"); err == nil {
		t.Fatal("unavailable package accepted")
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("failed download changed configuration")
	}
	e, ok := themepack.Lookup("heart")
	if !ok {
		t.Fatal("fixture missing")
	}
	b, err := os.ReadFile(filepath.Join("..", "..", "themes", e.Package))
	if err != nil {
		t.Fatal(err)
	}
	s, err := themepack.DefaultStore()
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Import(e, b); err != nil {
		t.Fatal(err)
	}
	if _, _, err := saveAppearance("theme", "heart"); err != nil {
		t.Fatal("offline activation:", err)
	}
	c, err := config.LoadDefault()
	if err != nil || c.Animation != "heart" || c.Style != "heart" || c.Background != e.Background {
		t.Fatalf("incomplete theme: %+v %v", c, err)
	}
	if animationByName("heart").FrameCount() != 2 {
		t.Fatal("installed artwork not loaded")
	}
}

func TestMissingAnimationFallsBackWithoutDownload(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	a := animationByName("slotmaschine")
	if a.FrameCount() != 96 {
		t.Fatal("missing package did not fall back to default")
	}
	s, _ := themepack.DefaultStore()
	if _, err := os.Stat(s.Directory); !os.IsNotExist(err) {
		t.Fatal("renderer wrote theme data")
	}
}
