package settingsui

import (
	"image/color"
	"strings"
	"testing"

	"github.com/reathloo/shellux/internal/config"
	"github.com/reathloo/shellux/internal/theme"
)

func sameColor(a, b color.Color) bool {
	ar, ag, ab, aa := a.RGBA()
	br, bg, bb, ba := b.RGBA()
	return ar == br && ag == bg && ab == bb && aa == ba
}

func TestSettingsFollowEveryHeaderStyle(t *testing.T) {
	for _, name := range config.Styles() {
		t.Run(name, func(t *testing.T) {
			c := config.Default()
			if err := c.ChangeAppearance("style", name); err != nil {
				t.Fatal(err)
			}
			m := newModel(c, func(config.Config) error { return nil })
			palette, err := theme.ByName(c.Style)
			if err != nil {
				t.Fatal(err)
			}
			s := m.styles()
			if !sameColor(s.accent.GetForeground(), palette.AccentColor) || !sameColor(s.focused.GetBackground(), palette.AccentColor) || !sameColor(s.border, palette.MutedColor) {
				t.Fatal("UI colors differ from header")
			}
			accentCode := strings.TrimSuffix(strings.TrimPrefix(palette.Accent, "\x1b["), "m")
			borderCode := strings.TrimSuffix(strings.TrimPrefix(palette.Muted, "\x1b["), "m")
			for _, focus := range []int{0, 1, 2, 3} {
				m.focus = focus
				view := m.View().Content
				if !strings.Contains(view, accentCode) || !strings.Contains(view, borderCode) {
					t.Fatal("view did not render style colors")
				}
			}
			m.openInput("Name", "theme", "my-theme", "")
			input := m.modal.input.Styles()
			if !sameColor(input.Cursor.Color, palette.AccentColor) || !sameColor(input.Focused.Prompt.GetForeground(), palette.AccentColor) || !sameColor(input.Focused.Placeholder.GetForeground(), palette.MutedColor) {
				t.Fatal("input retained unrelated colors")
			}
			checkBounds(t, m, [2]int{96, 30})
		})
	}
}

func TestStyleAndThemeSelectionRecolorsWithoutSaving(t *testing.T) {
	c := config.Default()
	saves := 0
	m := newModel(c, func(config.Config) error { saves++; return nil })
	before := m.View().Content
	m.section = 2
	chooseRow(t, m, "style", "")
	press(m, "enter")
	for i, choice := range m.modal.choices {
		if choice.value == "neon" {
			m.modal.selected = i
		}
	}
	press(m, "enter")
	neon, _ := theme.ByName("neon")
	if !sameColor(m.styles().border, neon.MutedColor) || m.View().Content == before {
		t.Fatal("style did not change immediately")
	}
	m.section = 1
	chooseRow(t, m, "theme", "orangedragon")
	press(m, "enter")
	orangedragon, _ := theme.ByName("orangedragon")
	if !sameColor(m.styles().focused.GetBackground(), orangedragon.AccentColor) {
		t.Fatal("theme did not update UI")
	}
	if m.original.Style != c.Style || saves != 0 {
		t.Fatal("recoloring persisted draft")
	}
	press(m, "esc")
	press(m, "right")
	press(m, "enter")
	if m.result != Cancelled || saves != 0 {
		t.Fatal("cancel saved preview")
	}
	reopened := newModel(c, func(config.Config) error { return nil })
	original, _ := theme.ByName(c.Style)
	if !sameColor(reopened.styles().border, original.MutedColor) {
		t.Fatal("cancel changed subsequent menu")
	}
}

func TestRainbowHeadingsUseAllSharedColors(t *testing.T) {
	c := config.Default()
	c.Style = "rainbow"
	m := newModel(c, func(config.Config) error { return nil })
	heading := m.styles().heading("EINSTELLUNGEN", 0)
	for _, color := range theme.RainbowColors() {
		// Compare rendered single-color fragments, independent of ANSI encoding.
		found := false
		for _, r := range "EINSTELLUNGEN" {
			s := m.styles().accent.Foreground(color).Render(string(r))
			if strings.Contains(heading, s) {
				found = true
				break
			}
		}
		if !found {
			t.Fatal("rainbow heading omits a header color")
		}
	}
}
