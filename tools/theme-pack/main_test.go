package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/reathloo/shellux/internal/themepack"
)

// Check published inputs as part of the normal suite, with no network or writes.
func TestRepositoryPackages(t *testing.T) {
	out := filepath.Join("..", "..", "themes")
	if err := verify(out); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(out, "catalog.json"))
	if err != nil {
		t.Fatal(err)
	}
	c, err := themepack.DecodeCatalog(b)
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Themes) == 0 {
		t.Fatal("missing theme packages")
	}
	s := themepack.Store{Directory: filepath.Join(out, "packages")}
	for _, e := range c.Themes {
		p, err := s.Load(e)
		if err != nil {
			t.Fatal(err)
		}
		if p.FrameCount() == 0 {
			t.Fatal("empty animation")
		}
	}
}

func TestBuildAndVersionProtection(t *testing.T) {
	out := t.TempDir()
	b, err := os.ReadFile(filepath.Join("..", "..", "themes", "packages", "heart-1.0.0.shellux-theme"))
	if err != nil {
		t.Fatal(err)
	}
	p, err := themepack.Decode(b)
	if err != nil {
		t.Fatal(err)
	}
	source := t.TempDir()
	m := p.Manifest
	if err := os.Mkdir(filepath.Join(source, "frames"), 0700); err != nil {
		t.Fatal(err)
	}
	for i, step := range m.Frames {
		if err := os.WriteFile(filepath.Join(source, step.File), []byte(p.Frame(i)), 0600); err != nil {
			t.Fatal(err)
		}
	}
	manifest, _ := json.Marshal(m)
	if err := os.WriteFile(filepath.Join(source, "theme.json"), manifest, 0600); err != nil {
		t.Fatal(err)
	}
	if err := build(source, out); err != nil {
		t.Fatal(err)
	}
	if err := build(source, out); err != nil {
		t.Fatal("identical rebuild failed:", err)
	}
	if err := verify(out); err != nil {
		t.Fatal(err)
	}
	catalogPath := filepath.Join(out, "catalog.json")
	before, _ := os.ReadFile(catalogPath)
	m.Background = "#123456"
	manifest, _ = json.Marshal(m)
	if err := os.WriteFile(filepath.Join(source, "theme.json"), manifest, 0600); err != nil {
		t.Fatal(err)
	}
	if err := build(source, out); err == nil {
		t.Fatal("replaced published version")
	}
	after, _ := os.ReadFile(catalogPath)
	if !bytes.Equal(before, after) {
		t.Fatal("catalog changed after rejected build")
	}
}
