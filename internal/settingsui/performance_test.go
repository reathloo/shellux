package settingsui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/reathloo/shellux/internal/config"
)

var benchmarkView tea.View

func BenchmarkSettingsView(b *testing.B) {
	for name, section := range map[string]int{
		"menu":       menuSection,
		"layouts":    presetsSection,
		"themes":     themesSection,
		"appearance": appearanceSection,
		"general":    generalSection,
	} {
		b.Run(name, func(b *testing.B) {
			m := benchmarkModel(section)
			benchmarkView = m.View()
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				benchmarkView = m.View()
			}
		})
	}
}

func BenchmarkSettingsNavigation(b *testing.B) {
	m := benchmarkModel(menuSection)
	down := tea.KeyPressMsg(tea.Key{Code: tea.KeyDown})
	up := tea.KeyPressMsg(tea.Key{Code: tea.KeyUp})
	benchmarkView = m.View()
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		m.Update(down)
		benchmarkView = m.View()
		m.Update(up)
		benchmarkView = m.View()
	}
}

func benchmarkModel(section int) *model {
	m := newModel(config.Default(), func(config.Config) error { return nil })
	m.width, m.height = 120, 36
	m.section, m.focus = section, 1
	m.checkInstalled = func(string) bool { return false }
	return m
}
