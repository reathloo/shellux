package playerui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/reathloo/shellux/internal/locale"
	"github.com/reathloo/shellux/internal/systeminfo"
	"github.com/reathloo/shellux/internal/theme"
)

func TestPlayerKeyboardControls(t *testing.T) {
	m := &model{width: 80, height: 24, language: locale.German}
	var called systeminfo.PlaybackAction
	m.control = func(action systeminfo.PlaybackAction) error { called = action; return nil }
	m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyRight}))
	if m.selected != 1 {
		t.Fatalf("selected = %d, want rewind", m.selected)
	}
	_, command := m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if command == nil || !m.busy {
		t.Fatal("enter did not start selected action")
	}
	if _, ok := command().(actionMsg); !ok || called != systeminfo.PlaybackRewind10 {
		t.Fatalf("called = %q, want rewind", called)
	}
	m.Update(actionMsg{})
	_, command = m.Update(tea.KeyPressMsg(tea.Key{Code: 'n', Text: "n"}))
	if command == nil {
		t.Fatal("shortcut did not start action")
	}
	command()
	if called != systeminfo.PlaybackNext {
		t.Fatalf("called = %q, want next", called)
	}
	m.Update(actionMsg{})
	_, command = m.Update(tea.KeyPressMsg(tea.Key{Code: 't', Text: "t"}))
	command()
	if called != systeminfo.PlaybackToggle {
		t.Fatalf("called = %q, want toggle", called)
	}
}

func TestPlayerViewSanitizesTrackMetadata(t *testing.T) {
	colors, err := theme.ByName("shelluxdefault")
	if err != nil {
		t.Fatal(err)
	}
	m := &model{width: 80, height: 24, colors: colors, language: locale.German,
		playback: systeminfo.Playback{Available: true, Title: "Track\x1b[2JName", Artist: "Artist\nName"}}
	view := ansi.Strip(m.View().Content)
	if strings.Contains(view, "\x1b") || !strings.Contains(view, "Track [2JName") || !strings.Contains(view, "Artist Name") {
		t.Fatalf("unsafe or missing metadata in view: %q", view)
	}
	for _, line := range strings.Split(view, "\n") {
		if ansi.StringWidth(line) > m.width {
			t.Fatalf("player exceeds terminal width: %q", line)
		}
	}
}
