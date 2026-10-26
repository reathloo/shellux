package settingsui

import "github.com/reathloo/shellux/internal/locale"

func (m *model) language() locale.Language {
	if m.resolvedLanguageValue != m.draft.Language || m.resolvedLanguage == "" {
		m.resolvedLanguageValue = m.draft.Language
		m.resolvedLanguage = locale.Resolve(m.draft.Language)
	}
	return m.resolvedLanguage
}
func (m *model) tr(text string) string { return m.language().Text(text) }

func (m *model) beginRowsFrame() {
	m.cacheRows = true
	m.invalidateRows()
}

func (m *model) endRowsFrame() {
	m.cacheRows = false
	m.invalidateRows()
}

func (m *model) invalidateRows() {
	m.cachedRows = nil
	m.cachedRowsSection = -1
}
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
