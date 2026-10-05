package config

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestConfigWriteReplacesFileAtomically(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	path, err := SetStyle("neon")
	if err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	reader, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	if _, err := SetStyle("purplemonster"); err != nil {
		t.Fatal(err)
	}
	// A reader that opened the old file must still see its complete contents.
	previous, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(previous, before) {
		t.Fatal("saving config modified the file an existing reader was using")
	}
	settings, err := Load(path)
	if err != nil || settings.Style != "purplemonster" {
		t.Fatalf("Load() after saving = %+v, %v", settings, err)
	}
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil || len(entries) != 1 {
		t.Fatalf("config directory after saving = %v, %v", entries, err)
	}
}

func TestConfigWritePreservesSymlink(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	path, err := SetStyle("neon")
	if err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(t.TempDir(), "shared-config.json")
	if err := os.Rename(path, target); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, path); err != nil {
		t.Fatal(err)
	}
	if savedPath, err := SetStyle("orb"); err != nil || savedPath != path {
		t.Fatalf("SetStyle() = %q, %v", savedPath, err)
	}
	if got, err := os.Readlink(path); err != nil || got != target {
		t.Fatalf("config symlink = %q, %v", got, err)
	}
	settings, err := Load(target)
	if err != nil || settings.Style != "orb" {
		t.Fatalf("Load(symlink target) = %+v, %v", settings, err)
	}
}

func TestConfigWriteDoesNotReplaceBrokenSymlink(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	path, err := DefaultPath()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(t.TempDir(), "missing-config.json")
	if err := os.Symlink(target, path); err != nil {
		t.Fatal(err)
	}
	if _, err := SetStyle("orb"); err == nil {
		t.Fatal("SetStyle() accepted an unresolved symlink")
	}
	if got, err := os.Readlink(path); err != nil || got != target {
		t.Fatalf("broken config symlink was replaced: %q, %v", got, err)
	}
}

func TestDefault(t *testing.T) {
	config := Default()
	if config.Animation != "shelluxdefault" {
		t.Fatalf("Default().Animation = %q, want shelluxdefault", config.Animation)
	}
	if config.Style != "shelluxdefault" || config.Background != "#000000" {
		t.Fatalf("default appearance = %+v, want the gold wordmark on black", config)
	}
	if !config.Visible["spotify"] || !config.Visible["progress"] || !config.Visible["uptime"] || config.Visible["temperature"] {
		t.Fatalf("Default().Visible = %v, want spotify, progress and uptime enabled and thermal disabled", config.Visible)
	}
	if err := config.Validate(); err != nil {
		t.Fatalf("Default().Validate() error = %v", err)
	}
}

func TestRemovedAnimationIsRejected(t *testing.T) {
	config := Default()
	config.Animation = "removed-animation"
	if err := config.Validate(); err == nil {
		t.Fatal("Validate() accepted removed animation")
	}
}

func TestLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "shellux.json")
	if err := os.WriteFile(path, []byte(`{"refresh_interval":"2s","visible":{"battery":false}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	config, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if config.Interval().String() != "2s" || config.Visible["battery"] || config.Visible["temperature"] != false || !config.Visible["cpu"] {
		t.Fatalf("Load() = %+v", config)
	}
}

func TestLoadMigratesLegacyThemeField(t *testing.T) {
	path := filepath.Join(t.TempDir(), "shellux.json")
	if err := os.WriteFile(path, []byte(`{"theme":"neon"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	config, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if config.Style != "neon" {
		t.Fatalf("Style = %q, want neon", config.Style)
	}
}

func TestSetDisplayWritesDefaultConfig(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	path, err := SetDisplay("thermal", "", true)
	if err != nil {
		t.Fatalf("SetDisplay() error = %v", err)
	}
	config, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !config.Visible["temperature"] {
		t.Fatalf("temperature visibility = false, want true")
	}
}

func TestSetAnimationWritesDefaultConfig(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	path, err := SetAnimation("purplemonster")
	if err != nil {
		t.Fatalf("SetAnimation() error = %v", err)
	}
	config, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if config.Animation != "purplemonster" {
		t.Fatalf("Animation = %q, want purplemonster", config.Animation)
	}
}

func TestSetAnimationAcceptsOrangeDragon(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	path, err := SetAnimation("orangedragon")
	if err != nil {
		t.Fatalf("SetAnimation() error = %v", err)
	}
	config, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if config.Animation != "orangedragon" {
		t.Fatalf("Animation = %q, want orangedragon", config.Animation)
	}
}

func TestSetStyleWritesDefaultConfig(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	path, err := SetStyle("neon")
	if err != nil {
		t.Fatalf("SetStyle() error = %v", err)
	}
	config, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if config.Style != "neon" {
		t.Fatalf("Style = %q, want neon", config.Style)
	}
}

func TestSetStyleAcceptsLuxuryAlias(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	path, err := SetStyle("luxury")
	if err != nil {
		t.Fatalf("SetStyle() error = %v", err)
	}
	config, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if config.Style != "dark-luxury" {
		t.Fatalf("Style = %q, want dark-luxury", config.Style)
	}
}

func TestSetStyleAcceptsAurora(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	path, err := SetStyle("aurora")
	if err != nil {
		t.Fatalf("SetStyle() error = %v", err)
	}
	config, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if config.Style != "aurora" {
		t.Fatalf("Style = %q, want aurora", config.Style)
	}
}

func TestSetStyleAcceptsRainbow(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	path, err := SetStyle("rainbow")
	if err != nil {
		t.Fatalf("SetStyle() error = %v", err)
	}
	config, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if config.Style != "rainbow" {
		t.Fatalf("Style = %q, want rainbow", config.Style)
	}
}

func TestSetStyleAcceptsOrangeDragon(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	path, err := SetStyle("orangedragon")
	if err != nil {
		t.Fatalf("SetStyle() error = %v", err)
	}
	config, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if config.Style != "orangedragon" {
		t.Fatalf("Style = %q, want orangedragon", config.Style)
	}
}

func TestSetStyleAcceptsPurpleMonster(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	path, err := SetStyle("purplemonster")
	if err != nil {
		t.Fatalf("SetStyle() error = %v", err)
	}
	config, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if config.Style != "purplemonster" {
		t.Fatalf("Style = %q, want purplemonster", config.Style)
	}
}

func TestInvalidConfig(t *testing.T) {
	config := Default()
	config.RefreshInterval = "0s"
	if err := config.Validate(); err == nil {
		t.Fatal("Validate() accepted zero refresh interval")
	}
}

func TestInvalidStyleIsRejected(t *testing.T) {
	config := Default()
	config.Style = "unknown"
	if err := config.Validate(); err == nil {
		t.Fatal("Validate() accepted an unsupported style")
	}
}

func TestSetThemeAppliesBuiltInTheme(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	selected, path, err := SetTheme("orangedragon")
	if err != nil {
		t.Fatalf("SetTheme() error = %v", err)
	}
	if selected != (SavedTheme{Animation: "orangedragon", Style: "orangedragon", Background: "#1e1210"}) {
		t.Fatalf("SetTheme() = %+v, want orangedragon theme", selected)
	}
	settings, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if settings.Animation != "orangedragon" || settings.Style != "orangedragon" {
		t.Fatalf("saved settings = %+v, want orangedragon theme", settings)
	}
}

func TestSetThemeAppliesOrbTheme(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	selected, path, err := SetTheme("orb")
	if err != nil {
		t.Fatalf("SetTheme(orb) error = %v", err)
	}
	if selected != (SavedTheme{Animation: "orb", Style: "orb", Background: "#0b1420"}) {
		t.Fatalf("SetTheme(orb) = %+v", selected)
	}
	settings, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if settings.Animation != "orb" || settings.Style != "orb" {
		t.Fatalf("saved settings = %+v", settings)
	}
}

func TestCustomThemeLifecycle(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if _, err := SetAnimation("orangedragon"); err != nil {
		t.Fatal(err)
	}
	if _, err := SetStyle("neon"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := SetBackground("#223344"); err != nil {
		t.Fatal(err)
	}
	name, _, err := SaveCurrentTheme("MY-NEON")
	if err != nil {
		t.Fatalf("SaveCurrentTheme() error = %v", err)
	}
	if name != "my-neon" {
		t.Fatalf("saved name = %q, want my-neon", name)
	}
	if _, _, err := SetTheme("default"); err != nil {
		t.Fatalf("SetTheme(default) error = %v", err)
	}
	selected, _, err := SetTheme("my-neon")
	if err != nil {
		t.Fatalf("SetTheme(custom) error = %v", err)
	}
	if selected != (SavedTheme{Animation: "orangedragon", Style: "neon", Background: "#223344"}) {
		t.Fatalf("custom selection = %+v", selected)
	}
	themes, err := Themes()
	if err != nil {
		t.Fatalf("Themes() error = %v", err)
	}
	for index, expected := range []string{"default", "anime-face", "heart", "liqudemetall", "loopingliqude", "orangedragon", "orb", "pinkcat", "purplemonster", "shelluxdefault", "slotmaschine", "spiderboy", "my-neon"} {
		if index >= len(themes) || themes[index].Name != expected {
			t.Fatalf("Themes() = %+v, expected %q at index %d", themes, expected, index)
		}
	}
	found := false
	for _, theme := range themes {
		if theme.Name == "my-neon" && theme.Custom && theme.Animation == "orangedragon" && theme.Style == "neon" && theme.Background == "#223344" {
			found = true
		}
	}
	if !found {
		t.Fatalf("Themes() = %+v, custom theme missing", themes)
	}
	if _, _, err := DeleteTheme("my-neon"); err != nil {
		t.Fatalf("DeleteTheme() error = %v", err)
	}
	if _, _, err := SetTheme("my-neon"); err == nil {
		t.Fatal("deleted custom theme can still be selected")
	}
}

func TestThemeOperationsRejectReservedNames(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if _, _, err := SaveCurrentTheme("purplemonster"); err == nil {
		t.Fatal("SaveCurrentTheme() accepted built-in name")
	}
	if _, _, err := DeleteTheme("orangedragon"); err == nil {
		t.Fatal("DeleteTheme() accepted built-in theme")
	}
}

func TestSetThemeRandomUsesAvailableValues(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	selected, _, err := SetTheme("random")
	if err != nil {
		t.Fatalf("SetTheme(random) error = %v", err)
	}
	if !contains(Animations(), selected.Animation) || !contains(Styles(), DisplayStyle(selected.Style)) {
		t.Fatalf("random selection = %+v, want supported values", selected)
	}
}

func TestThemesRestoreAllThreeComponents(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, name := range []string{"purplemonster", "orangedragon", "liqudemetall", "orb", "anime-face", "loopingliqude", "heart", "pinkcat", "spiderboy", "shelluxdefault", "slotmaschine", "default"} {
		t.Run(name, func(t *testing.T) {
			if _, _, err := SetBackground("#abcdef"); err != nil {
				t.Fatal(err)
			}
			selected, _, err := SetTheme(name)
			if err != nil {
				t.Fatal(err)
			}
			settings, err := LoadDefault()
			if err != nil {
				t.Fatal(err)
			}
			if settings.Animation != selected.Animation || settings.Style != selected.Style || settings.Background != selected.Background {
				t.Fatalf("theme only partially applied: %+v, want %+v", settings, selected)
			}
			if selected.Background == "#abcdef" || selected.Background == "" {
				t.Fatal("theme did not restore its background")
			}
		})
	}
}

func TestLoadMigratesThemesWithoutBackground(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	path, err := SetStyle("neon")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"animation":"orb","themes":{"old-look":{"animation":"orangedragon","style":"neon"}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	settings, err := LoadDefault()
	if err != nil {
		t.Fatal(err)
	}
	if settings.Background != "default" || settings.Themes["old-look"].Background != "default" {
		t.Fatalf("legacy background was not preserved: %+v", settings)
	}
	if _, _, err := SetBackground("navy"); err != nil {
		t.Fatal(err)
	}
	selected, _, err := SetTheme("old-look")
	if err != nil || selected.Background != "default" {
		t.Fatalf("legacy theme = %+v, %v", selected, err)
	}
}

func TestInvalidSavedThemeBackgroundIsRejected(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	path, err := SetStyle("neon")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"themes":{"broken":{"animation":"purplemonster","style":"neon","background":"invalid"}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadDefault(); err == nil {
		t.Fatal("invalid saved background accepted")
	}
}

func TestRandomThemeUsesMatchingAnimationBackground(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	selected, _, err := SetTheme("random")
	if err != nil {
		t.Fatal(err)
	}
	if selected.Background != availableThemes()[selected.Animation].Background {
		t.Fatalf("random theme background does not match animation: %+v", selected)
	}
}
