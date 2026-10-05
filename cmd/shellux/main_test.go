package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/reathloo/shellux/internal/config"
	"github.com/reathloo/shellux/internal/render"
	"github.com/reathloo/shellux/internal/systeminfo"
	"github.com/reathloo/shellux/internal/theme"
)

func TestHeaderGutterIsEmptyWithoutDashboard(t *testing.T) {
	for _, size := range [][2]int{{140, 2}, {3, 40}} {
		layout := render.NewHeaderLayout(systeminfo.Snapshot{}, "animation", theme.DarkLuxury(), nil).FitTerminal(size[0], size[1])
		if got := headerGutter(layout); got != "" {
			t.Fatalf("headerGutter() at %dx%d = %q, want no terminal writes", size[0], size[1], got)
		}
	}
}

func TestStatusReportNamesMatchingTheme(t *testing.T) {
	settings := config.Default()
	themes := []config.ThemeInfo{
		{Name: "default", Animation: "shelluxdefault", Style: "shelluxdefault", Background: settings.Background},
		{Name: "purplemonster", Animation: "purplemonster", Style: "purplemonster", Background: settings.Background},
	}
	report := statusReport(settings, themes)
	for _, expected := range []string{"Theme match:     default", "Animation:       shelluxdefault", "Style:           shelluxdefault", "Metrics refresh: 1s", "Animation frame:"} {
		if !strings.Contains(report, expected) {
			t.Fatalf("statusReport() = %q, missing %q", report, expected)
		}
	}
	settings.Background = "#123456"
	if report := statusReport(settings, themes); !strings.Contains(report, "Theme match:     custom combination") || !strings.Contains(report, "Background:      #123456") {
		t.Fatalf("background-only change was ignored: %q", report)
	}
	settings.Background = themes[0].Background
	settings.Style = "neon"
	if report := statusReport(settings, themes); !strings.Contains(report, "Theme match:     custom combination") {
		t.Fatalf("statusReport() = %q, want custom combination", report)
	}
}

func TestStatusReportNamesBackgroundIndependentlyOfAnimation(t *testing.T) {
	settings := config.Default()
	for _, test := range []struct{ color, name string }{
		{"#140d20", "purplemonster"}, {"#1e1210", "orangedragon"}, {"#0b1420", "orb"}, {"#11131c", "liqudemetall"}, {"#100317", "anime-face"}, {"#17191f", "loopingliqude"}, {"#1b0d15", "heart"}, {"#24121d", "pinkcat"}, {"#0c1018", "spiderboy"}, {"#000000", "shelluxdefault"}, {"#100d12", "slotmaschine"},
		{"#000080", "navy"}, {"default", "default"}, {"#123456", "#123456"},
	} {
		settings.Background = test.color
		report := statusReport(settings, nil)
		if !strings.Contains(report, "Background:      "+test.name+"\n") {
			t.Fatalf("statusReport() for %q = %q, want background %q", test.color, report, test.name)
		}
	}
}

func TestDoctorReport(t *testing.T) {
	directory := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", directory)
	t.Setenv("SHELLUX_DIAG_SHELL", "zsh")
	t.Setenv("SHELLUX_DIAG_ENABLED", "1")
	t.Setenv("SHELLUX_DIAG_WATCH", "1")
	var output bytes.Buffer
	if !doctorReport(&output) || !strings.Contains(output.String(), "Renderer:    OK") {
		t.Fatalf("doctorReport() = %q, want healthy renderer", output.String())
	}
	t.Setenv("SHELLUX_DIAG_WATCH", "paused")
	output.Reset()
	if !doctorReport(&output) || !strings.Contains(output.String(), "Renderer:    paused") {
		t.Fatalf("doctorReport() = %q, want intentional pause", output.String())
	}
	t.Setenv("SHELLUX_DIAG_ENABLED", "0")
	t.Setenv("SHELLUX_DIAG_WATCH", "0")
	output.Reset()
	if !doctorReport(&output) || !strings.Contains(output.String(), "Header:      off") {
		t.Fatalf("doctorReport() = %q, want intentionally disabled header", output.String())
	}
	path := filepath.Join(directory, "shellux", "config.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("{invalid"), 0o600); err != nil {
		t.Fatal(err)
	}
	output.Reset()
	if doctorReport(&output) || !strings.Contains(output.String(), "Config:      ERROR") {
		t.Fatalf("doctorReport() = %q, want config error", output.String())
	}
}

func TestMenuReportUsesPublicNamesAndComponentState(t *testing.T) {
	settings := config.Default()
	settings.Visible["cpu.bar"] = false
	report := menuReport(settings)
	lines := strings.Split(report, "\n")
	for _, name := range config.MenuItems() {
		public := config.DisplayMenuItem(name)
		count := 0
		for _, line := range lines {
			if strings.HasPrefix(strings.TrimSpace(line), public+" ") {
				count++
			}
		}
		if count != 1 {
			t.Fatalf("menu contains %d rows for %s", count, public)
		}
	}
	if !strings.Contains(report, "cpu              on  bar: off  percent: on  value: on") {
		t.Fatalf("missing CPU component state: %q", report)
	}
	if !strings.Contains(report, "traffic          on  upload: on  download: on  latency: off") {
		t.Fatalf("missing Traffic component state: %q", report)
	}
	for _, old := range []string{"directory", "network-traffic", "network-status", "temperature", "playback", "ram-free", "disk-free", "upload", "download", "latency"} {
		for _, line := range lines {
			if strings.HasPrefix(strings.TrimSpace(line), old+" ") {
				t.Fatalf("obsolete public name %s in report", old)
			}
		}
	}
}
