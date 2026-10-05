package settingsui

import (
	"image/color"
	"strings"

	"charm.land/bubbles/v2/textinput"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/reathloo/shellux/internal/theme"
)

// uiStyles follows the draft so a style/theme selection is immediately visible.
// Terminal backgrounds and the saved configuration still change only on save.
type uiStyles struct {
	accent, secondary, muted, focused, normal lipgloss.Style
	border, cursor                            color.Color
	rainbow                                   []ansi.Color
}

func (m *model) styles() uiStyles {
	palette, err := theme.ByName(m.draft.Style)
	if err != nil {
		palette = theme.DarkLuxury()
	}
	styles := uiStyles{
		accent:    lipgloss.NewStyle().Foreground(palette.AccentColor).Bold(true),
		secondary: lipgloss.NewStyle().Foreground(palette.SecondaryColor),
		muted:     lipgloss.NewStyle().Foreground(palette.MutedColor),
		normal:    lipgloss.NewStyle(),
		// All built-in accents are bright; dark text keeps selected controls legible.
		focused: lipgloss.NewStyle().Foreground(lipgloss.Color("#111111")).Background(palette.AccentColor).Bold(true),
		border:  palette.MutedColor, cursor: palette.AccentColor,
	}
	if palette.Rainbow {
		styles.rainbow = theme.RainbowColors()
	}
	return styles
}

func (s uiStyles) heading(text string, offset int) string {
	if len(s.rainbow) == 0 {
		return s.accent.Render(text)
	}
	var result strings.Builder
	for _, r := range text {
		if r == ' ' {
			result.WriteRune(r)
			continue
		}
		result.WriteString(lipgloss.NewStyle().Foreground(s.rainbow[offset%len(s.rainbow)]).Bold(true).Render(string(r)))
		offset++
	}
	return result.String()
}

func (s uiStyles) input() textinput.Styles {
	styles := textinput.DefaultDarkStyles()
	state := textinput.StyleState{Text: s.normal, Prompt: s.accent, Placeholder: s.muted, Suggestion: s.muted}
	styles.Focused, styles.Blurred = state, state
	styles.Cursor.Color = s.cursor
	return styles
}
