package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestCodingProfileManualAndAutomaticSelection(t *testing.T) {
	root := t.TempDir()
	project := filepath.Join(root, "project")
	nested := filepath.Join(project, "src", "module")
	if err := os.MkdirAll(nested, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "go.mod"), []byte("module example.test/project\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, ".git"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	base := Default()
	base.Style = "aurora"
	base.Background = "#123456"
	original := base.Clone()
	if base.ActiveProfile(nested) != "" || base.ActiveProfile(root) != "" {
		t.Fatal("manual standard selected Coding")
	}
	if err := base.SetProfileSelection("auto"); err != nil {
		t.Fatal(err)
	}
	if base.ActiveProfile(nested) != "coding" || base.ActiveProfile(root) != "" {
		t.Fatal("automatic project marker detection failed")
	}
	effective := base.EffectiveProfile(nested)
	if effective.Style != base.Style || effective.Animation != base.Animation || effective.Background != base.Background || MenuSlots(effective.Visible) > 9 {
		t.Fatalf("Coding changed appearance or exceeded slots: %+v", effective)
	}
	if effective.MenuLayout.Groups[0] != "CODING" || !Shown(effective.Visible, "git-branch") || Shown(effective.Visible, "spotify") || Shown(effective.Visible, "cpu.top") || !Shown(effective.Visible, "time") {
		t.Fatal("Coding layout is not compact and project-focused")
	}
	if err := effective.Validate(); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(base.Visible, original.Visible) || !reflect.DeepEqual(base.MenuLayout, original.MenuLayout) || base.Style != original.Style {
		t.Fatal("profile overlay changed saved base settings")
	}
	if err := base.SelectProfile("coding"); err != nil {
		t.Fatal(err)
	}
	if base.ActiveProfile(root) != "coding" || base.EffectiveProfile(root).Style != base.Style {
		t.Fatal("manual Coding did not apply outside a project")
	}
}

func TestGitDirectoryAndOtherProjectMarkers(t *testing.T) {
	for _, marker := range []string{".git", "package.json", "pyproject.toml", "Cargo.toml", "pom.xml", "build.gradle.kts"} {
		t.Run(marker, func(t *testing.T) {
			project := t.TempDir()
			if err := os.WriteFile(filepath.Join(project, marker), nil, 0600); err != nil {
				t.Fatal(err)
			}
			if !isCodingProject(project) {
				t.Fatal("project marker not detected")
			}
		})
	}
}

func TestCodingProfileShowsAvailableToolsAndCommandResult(t *testing.T) {
	bin := t.TempDir()
	for _, name := range []string{"go", "docker"} {
		if err := os.WriteFile(filepath.Join(bin, name), []byte("#!/bin/sh\nexit 0\n"), 0700); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", bin)
	t.Setenv("SHELLUX_LAST_COMMAND_STATUS", "")
	t.Setenv("SHELLUX_LAST_COMMAND_SECONDS", "")
	base := Default()
	if err := base.SelectProfile("coding"); err != nil {
		t.Fatal(err)
	}
	effective := base.EffectiveProfile(t.TempDir())
	for _, key := range []string{"runtimes", "runtimes.go", "dev-context", "dev-context.docker"} {
		if !Shown(effective.Visible, key) {
			t.Fatalf("available %s hidden", key)
		}
	}
	for _, key := range []string{"runtimes.node", "dev-context.kubernetes", "last-command"} {
		if Shown(effective.Visible, key) {
			t.Fatalf("unavailable %s shown", key)
		}
	}
	if Shown(effective.Visible, "git-branch") || Shown(effective.Visible, "git-sync") {
		t.Fatal("Git rows shown outside a Git repository")
	}
	t.Setenv("SHELLUX_LAST_COMMAND_STATUS", "1")
	t.Setenv("SHELLUX_LAST_COMMAND_SECONDS", "2")
	if !Shown(base.EffectiveProfile(t.TempDir()).Visible, "last-command") {
		t.Fatal("last command row missing when shell data is available")
	}
}
