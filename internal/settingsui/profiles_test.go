package settingsui

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
	"github.com/reathloo/shellux/internal/config"
)

func TestProfileSelectionStaysInDraft(t *testing.T) {
	base := config.Default()
	var saved config.Config
	m := newModel(base, func(c config.Config) error { saved = c; return nil })
	m.section = presetsSection
	if got := len(m.rows()); got != 3+len(m.draft.MenuPresetOptions())+len(config.ProfileNames) {
		t.Fatalf("profile rows = %d", got)
	}
	chooseRow(t, m, "profile-selection", "")
	press(m, "space")
	if m.draft.ProfileSelection != "auto" {
		t.Fatal("automatic selection not set")
	}
	chooseRow(t, m, "profile", "coding")
	press(m, "enter")
	if m.draft.ProfileSelection != "manual" {
		t.Fatal("manual profile choice did not leave automatic selection")
	}
	press(m, "enter")
	if m.draft.SelectedProfile != "" {
		t.Fatal("selected profile did not toggle off")
	}
	press(m, "enter")
	if m.draft.SelectedProfile != "coding" || base.SelectedProfile != "" || base.ProfileSelection != "manual" {
		t.Fatal("profile selection or switch escaped the draft")
	}
	chooseRow(t, m, "profile-selection", "")
	press(m, "space")
	if !m.dirty() {
		t.Fatal("profile settings did not mark draft unsaved")
	}
	press(m, "ctrl+s")
	if m.result != Saved || saved.SelectedProfile != "coding" || saved.ProfileSelection != "auto" {
		t.Fatal("profile choices were not saved")
	}

	m = newModel(base, func(config.Config) error { t.Fatal("cancel saved profile settings"); return nil })
	m.section = presetsSection
	chooseRow(t, m, "profile-selection", "")
	press(m, "enter")
	press(m, "esc")
	press(m, "right")
	press(m, "enter")
	if m.result != Cancelled {
		t.Fatal("cancel did not discard profile changes")
	}
}

func TestFocusTimerStartsWhenSettingsAreSaved(t *testing.T) {
	base := config.Default()
	base.Style, base.Background = "aurora", "#123456"
	if err := base.SelectProfile("coding"); err != nil {
		t.Fatal(err)
	}
	var saved config.Config
	m := newModel(base, func(c config.Config) error { saved = c; return nil })
	m.section = presetsSection
	chooseRow(t, m, "profile", "focus")
	press(m, "enter")
	if m.modal == nil || m.modal.field != "focus-duration" || m.draft.SelectedProfile != "coding" {
		t.Fatal("Focus did not offer a timer before changing the draft")
	}
	press(m, "enter") // Default: 30 minutes.
	if m.draft.SelectedProfile != "focus" || m.draft.FocusMinutes != 30 || m.draft.FocusUntil != "" {
		t.Fatal("Focus duration was not staged")
	}
	before := time.Now()
	press(m, "ctrl+s")
	if m.result != Saved || saved.FocusUntil == "" || saved.FocusPreviousProfile != "coding" {
		t.Fatal("Focus session was not saved")
	}
	if remaining := saved.FocusRemaining(before); remaining < 29*time.Minute || remaining > 31*time.Minute {
		t.Fatalf("saved focus remaining = %s", remaining)
	}
	if saved.Background != "#123456" || saved.Style != "aurora" {
		t.Fatal("Focus overwrote the saved theme")
	}
}

func TestGermanProfileRowsFitTerminal(t *testing.T) {
	c := config.Default()
	c.Language = "de"
	m := newModel(c, func(config.Config) error { return nil })
	m.section = presetsSection
	chooseRow(t, m, "profile", "presentation")
	view := ansi.Strip(m.View().Content)
	if !strings.Contains(view, "Preset-Vorlagen") || !strings.Contains(view, "Präsentation") || !strings.Contains(view, "Auswahl") {
		t.Fatalf("German profile UI missing: %s", view)
	}
	for _, size := range [][2]int{{96, 30}, {64, 18}} {
		m.width, m.height = size[0], size[1]
		for _, line := range strings.Split(m.View().Content, "\n") {
			if ansi.StringWidth(line) > size[0] {
				t.Fatalf("profile UI exceeds %d columns", size[0])
			}
		}
	}
}

func TestPresetLayoutColumnsAndThemeIndependence(t *testing.T) {
	c := config.Default()
	c.Style = "aurora"
	c.Background = "#123456"
	m := newModel(c, func(config.Config) error { return nil })
	m.section, m.focus = presetsSection, 1
	view := ansi.Strip(m.View().Content)
	if !strings.Contains(view, "Saved layouts") || !strings.Contains(view, "Preset layouts") || !strings.Contains(view, "Selection  · Manual") {
		t.Fatalf("layout columns or selection missing: %s", view)
	}
	chooseRow(t, m, "preset", "default")
	press(m, "right")
	if m.selectedRow().action != "profile" {
		t.Fatal("right arrow did not enter preset layout column")
	}
	chooseRow(t, m, "profile", "coding")
	press(m, "enter")
	if m.draft.Style != "aurora" || m.draft.Background != "#123456" || m.draft.SelectedProfile != "coding" {
		t.Fatal("Coding preset changed appearance or failed to select")
	}
	chooseRow(t, m, "profile", "minimal")
	press(m, "enter")
	if m.draft.Style != "aurora" || m.draft.Background != "#123456" || m.draft.SelectedProfile != "minimal" {
		t.Fatal("Minimal preset changed appearance or failed to select")
	}
	chooseRow(t, m, "profile", "presentation")
	press(m, "enter")
	if m.draft.Style != "aurora" || m.draft.Background != "#123456" || m.draft.SelectedProfile != "presentation" {
		t.Fatal("Presentation preset changed appearance or failed to select")
	}
	chooseRow(t, m, "preset", "default")
	press(m, "enter")
	if m.draft.ProfileSelection != "manual" || m.draft.SelectedProfile != "" || m.draft.Style != "aurora" || m.draft.Background != "#123456" {
		t.Fatal("saved/default layout did not leave Coding while preserving appearance")
	}
}
