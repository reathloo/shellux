package settingsui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/reathloo/shellux/internal/config"
)

// Locate the actual rendered control instead of repeating the layout formula.
func clickRendered(t *testing.T, m *model, rowText, control string) {
	t.Helper()
	for y, rendered := range strings.Split(m.View().Content, "\n") {
		plain := ansi.Strip(rendered)
		if !strings.Contains(plain, rowText) {
			continue
		}
		if index := strings.Index(plain, control); index >= 0 {
			x := ansi.StringWidth(plain[:index])
			m.Update(tea.MouseClickMsg(tea.Mouse{X: x, Y: y, Button: tea.MouseLeft}))
			return
		}
	}
	t.Fatalf("control %q in row %q not visible", control, rowText)
}

func TestMouseFollowsScrolledNavigation(t *testing.T) {
	m := newModel(config.Default(), func(config.Config) error { return nil })
	m.width, m.height = 64, 18
	m.section = doctorSection
	clickRendered(t, m, "General", "General")
	if m.section != generalSection {
		t.Fatalf("clicked General, selected section %d", m.section)
	}
}

func TestMouseUsesVisibleSwitchesWhenContentIsUnfocused(t *testing.T) {
	m := newModel(config.Default(), func(config.Config) error { return nil })
	m.width, m.height = 64, 25
	chooseRow(t, m, "toggle", "cpu")
	m.part, m.focus = 5, 0
	clickRendered(t, m, "cpu", "[✓] Enabled")
	if m.part != 1 || config.Shown(m.draft.Visible, "cpu") {
		t.Fatal("click did not toggle the visible Enabled switch")
	}
}

func TestMouseMatchesRenderedPanelAtDifferentSizes(t *testing.T) {
	for _, size := range [][2]int{{64, 18}, {65, 19}, {95, 29}, {96, 30}, {141, 41}} {
		m := newModel(config.Default(), func(config.Config) error { return nil })
		m.width, m.height = size[0], size[1]
		clickRendered(t, m, "Appearance", "Appearance")
		if m.section != appearanceSection {
			t.Fatalf("Appearance click missed at %v", size)
		}
		clickRendered(t, m, "Background", "Background")
		if m.modal == nil || m.modal.field != "background" {
			t.Fatalf("Background click missed at %v", size)
		}
	}
}

func mouseAt(m *model, x, y int, button tea.MouseButton, wheel bool) {
	bw, bh := min(96, m.width-2), min(30, m.height-2)
	if m.modal != nil && m.modal.kind == "select" && m.modal.field == "animation" {
		bh = min(36, m.height-2)
	}
	mouse := tea.Mouse{X: (m.width-bw)/2 + 2 + x, Y: (m.height-bh)/2 + 1 + y, Button: button}
	if wheel {
		m.Update(tea.MouseWheelMsg(mouse))
	} else {
		m.Update(tea.MouseClickMsg(mouse))
	}
}

func TestMouseNavigatesAndTogglesMenuControls(t *testing.T) {
	m := newModel(config.Default(), func(config.Config) error { return nil })
	if m.View().MouseMode != tea.MouseModeCellMotion {
		t.Fatal("mouse reporting is not enabled")
	}
	// The first visible content row is the platform switch.
	row := m.rows()[1]
	partX := 21 + 11
	mouseAt(m, partX, 3, tea.MouseLeft, false)
	if m.focus != 1 || m.cursors[menuSection] != 1 || m.part != 1 || config.Shown(m.draft.Visible, row.entry.Key) {
		t.Fatal("click did not toggle the enabled switch")
	}
	// A click on the Appearance navigation entry changes section.
	mouseAt(m, 3, 2+appearanceSection, tea.MouseLeft, false)
	if m.section != appearanceSection || m.focus != 1 {
		t.Fatal("navigation click was ignored")
	}
	mouseAt(m, 25, 5, tea.MouseWheelDown, true)
	if m.cursors[appearanceSection] != 1 {
		t.Fatal("wheel did not move the content selection")
	}
}

func TestMouseAppliesAppearanceChoiceAndSaves(t *testing.T) {
	var saved config.Config
	m := newModel(config.Default(), func(c config.Config) error { saved = c; return nil })
	m.section = appearanceSection
	chooseRow(t, m, "style", "")
	m.activate()
	if m.modal == nil || m.modal.kind != "select" {
		t.Fatal("style selector did not open")
	}
	for i, option := range m.modal.choices {
		if option.value == "neon" {
			mouseAt(m, 4, i+2, tea.MouseLeft, false)
			break
		}
	}
	if m.draft.Style != "neon" || m.modal == nil {
		t.Fatal("click did not preview style inside selector")
	}
	press(m, "esc")
	// Save lives below the content body.
	mouseAt(m, 3, (min(30, m.height-2)-2)-9+6, tea.MouseLeft, false)
	if !m.done || m.result != Saved || saved.Style != "neon" {
		t.Fatal("save click did not persist the selected style")
	}
}

func TestMouseTogglesVisibleDetailSwitch(t *testing.T) {
	m := newModel(config.Default(), func(config.Config) error { return nil })
	chooseRow(t, m, "toggle", "cpu")
	selected := m.cursors[menuSection]
	start, _ := window(len(m.rows()), selected, (min(30, m.height-2)-2)-9)
	row := m.selectedRow()
	if len(row.entry.Parts) == 0 {
		t.Fatal("CPU has no detail switches")
	}
	key := row.entry.Key + "." + row.entry.Parts[0]
	before := config.Shown(m.draft.Visible, key)
	// The first detail cell follows the eight-character label and Enabled cell.
	mouseAt(m, 21+23, 2+selected-start, tea.MouseLeft, false)
	if m.part != 2 || config.Shown(m.draft.Visible, key) == before {
		t.Fatal("detail switch was not toggled")
	}
}

func TestMouseSelectsPresetFromRightColumn(t *testing.T) {
	m := newModel(config.Default(), func(config.Config) error { return nil })
	m.section = presetsSection
	// The right column starts after the Saved layouts column and its gutter.
	rw := (min(96, m.width-2) - 4) - 21
	rightX := 21 + (rw-2)/2 + 3
	for i, name := range config.ProfileNames {
		if name == "coding" {
			mouseAt(m, rightX, 4+1+i, tea.MouseLeft, false)
			break
		}
	}
	if m.draft.SelectedProfile != "coding" || m.focus != 1 {
		t.Fatal("right-column preset click did not select Coding")
	}
}
