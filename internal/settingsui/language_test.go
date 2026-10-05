package settingsui

import (
	"github.com/charmbracelet/x/ansi"
	"github.com/reathloo/shellux/internal/config"
	"strings"
	"testing"
)

func TestLanguageSelectionSaveAndCancel(t *testing.T) {
	t.Setenv("LC_ALL", "C")
	original := config.Default()
	saved := original
	m := newModel(original, func(c config.Config) error { saved = c; return nil })
	m.section = 3
	chooseRow(t, m, "language", "")
	press(m, "enter")
	press(m, "down")
	press(m, "down")
	press(m, "enter")
	if m.draft.Language != "de" || saved.Language != "auto" {
		t.Fatal("language selection must stay in draft")
	}
	view := ansi.Strip(m.View().Content)
	for _, word := range []string{"Allgemein", "Sprache", "Speichern"} {
		if !strings.Contains(view, word) {
			t.Fatalf("missing %s: %s", word, view)
		}
	}
	press(m, "ctrl+s")
	if saved.Language != "de" || m.result != Saved {
		t.Fatal("language not saved")
	}
	m = newModel(original, func(c config.Config) error { t.Fatal("cancel saved"); return nil })
	m.draft.Language = "de"
	press(m, "esc")
	if m.modal == nil || m.modal.selected != 0 {
		t.Fatal("discard must default to editing")
	}
	press(m, "right")
	press(m, "enter")
	if m.result != Cancelled {
		t.Fatal("cancel failed")
	}
}
func TestGermanSmallWindowAndLayout(t *testing.T) {
	c := config.Default()
	c.Language = "de"
	m := newModel(c, nil)
	for _, size := range [][2]int{{96, 30}, {64, 18}, {50, 12}} {
		m.width, m.height = size[0], size[1]
		v := m.View().Content
		for _, line := range strings.Split(v, "\n") {
			if ansi.StringWidth(line) > size[0] {
				t.Fatalf("overflow at %v: %q", size, line)
			}
		}
		if size[0] < 64 && !strings.Contains(v, "vergrößern") {
			t.Fatal("small-window message not translated")
		}
	}
}
