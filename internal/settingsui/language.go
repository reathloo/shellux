package settingsui

import "github.com/reathloo/shellux/internal/locale"

func (m *model) language() locale.Language { return locale.Resolve(m.draft.Language) }
func (m *model) tr(text string) string     { return m.language().Text(text) }
func languageName(value string) string {
	switch value {
	case "de":
		return "Deutsch"
	case "en":
		return "English"
	default:
		return "Automatic (system)"
	}
}
