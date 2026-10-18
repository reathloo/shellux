package settingsui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// mousePosition translates terminal coordinates into the bordered panel.
func (m *model) mousePosition(mouse tea.Mouse) (x, y, width, height int, inside bool) {
	if m.small() {
		return 0, 0, 0, 0, false
	}
	bw, bh := min(96, m.width-2), min(30, m.height-2)
	if m.modal != nil && m.modal.kind == "select" && m.modal.field == "animation" {
		bh = min(36, m.height-2)
	}
	width, height = bw-4, bh-2
	x = mouse.X - (m.width-bw)/2 - 2
	y = mouse.Y - (m.height-bh)/2 - 1
	return x, y, width, height, x >= 0 && x < width && y >= 0 && y < height
}

func (m *model) updateMouse(msg tea.Msg) tea.Cmd {
	if m.doctorBusy || m.downloading || m.moveMode {
		return nil
	}
	var mouse tea.Mouse
	click := false
	switch event := msg.(type) {
	case tea.MouseClickMsg:
		mouse, click = event.Mouse(), true
		if mouse.Button != tea.MouseLeft {
			return nil
		}
	case tea.MouseWheelMsg:
		mouse = event.Mouse()
		if mouse.Button != tea.MouseWheelUp && mouse.Button != tea.MouseWheelDown {
			return nil
		}
	}
	x, y, width, height, inside := m.mousePosition(mouse)
	if !inside {
		return nil
	}
	if m.modal != nil {
		return m.mouseDialog(x, y, width, height, mouse, click)
	}
	bodyHeight := height - 9
	if click && y == bodyHeight+6 {
		saveWidth := lipgloss.Width(m.tr("[ Save ]"))
		if x < saveWidth {
			m.focus = 2
			return m.saveDraft()
		}
		if x >= saveWidth+2 && x < saveWidth+2+lipgloss.Width(m.tr("[ Cancel ]")) {
			m.focus = 3
			return m.cancel()
		}
		return nil
	}
	if y < 2 || y >= 2+bodyHeight {
		return nil
	}
	if x < 18 {
		if !click {
			m.focus = 0
			key := tea.KeyDown
			if mouse.Button == tea.MouseWheelUp {
				key = tea.KeyUp
			}
			_, cmd := m.Update(tea.KeyPressMsg(tea.Key{Code: key}))
			return cmd
		}
		if click {
			start, end := window(len(sections), m.section, min(bodyHeight, len(sections)))
			index := start + y - 2
			if index < end {
				m.section, m.focus, m.part = index, 1, 0
				if index == themesSection && m.refresh && !m.refreshStarted {
					return m.refreshCatalog()
				}
				if index == doctorSection && m.doctorChecks == nil {
					return m.runDoctorScan()
				}
			}
		}
		return nil
	}
	if x < 21 {
		return nil
	}
	rows := m.rows()
	selected := min(m.cursors[m.section], len(rows)-1)
	if !click {
		step := 1
		if mouse.Button == tea.MouseWheelUp {
			step = -1
		}
		m.focus = 1
		m.moveRow(step)
		return nil
	}
	rowIndex := y - 2
	if m.section == presetsSection {
		if rowIndex == 0 {
			rowIndex = 0
		} else if rowIndex == 1 {
			return nil
		} else {
			rw := width - 21
			leftWidth := (rw - 2) / 2
			leftEnd := 2 + len(m.draft.MenuPresetOptions())
			visible := bodyHeight - 2
			if x-21 < leftWidth {
				leftSelected := 0
				if selected > 0 && selected < leftEnd {
					leftSelected = selected - 1
				}
				start, _ := window(leftEnd-1, leftSelected, visible)
				rowIndex = 1 + start + rowIndex - 2
				if rowIndex >= leftEnd {
					return nil
				}
			} else if x-21 >= leftWidth+2 {
				rightSelected := 0
				if selected >= leftEnd {
					rightSelected = selected - leftEnd
				}
				start, _ := window(len(rows)-leftEnd, rightSelected, visible)
				rowIndex = leftEnd + start + rowIndex - 2
			} else {
				return nil
			}
		}
	} else {
		start, _ := window(len(rows), selected, bodyHeight)
		rowIndex += start
	}
	if rowIndex < 0 || rowIndex >= len(rows) {
		return nil
	}
	// Hit-test the controls as they were drawn, before changing focus.
	part, hitSwitch := 0, false
	if rows[rowIndex].action == "toggle" {
		part, hitSwitch = m.switchHit(rows[rowIndex], x-21, width-21, rowIndex == selected && m.focus == 1)
	}
	m.cursors[m.section], m.focus = rowIndex, 1
	r := rows[rowIndex]
	if r.action == "toggle" {
		if hitSwitch {
			m.part = part
			return m.activate()
		}
		m.part = 0
		return nil
	}
	m.part = 0
	if r.heading {
		return nil
	}
	return m.activate()
}

// switchHit follows the same horizontal window and cell widths as switchRow.
func (m *model) switchHit(r row, x, width int, focus bool) (int, bool) {
	labelWidth := lipgloss.Width("› " + fmt.Sprintf("%-8s", m.tr(r.label)))
	master := m.tr("Enabled")
	if len(r.entry.Parts) == 0 {
		master += " · " + m.entryPreview(r.entry.Key)
	}
	labels := []string{master}
	for _, part := range r.entry.Parts {
		labels = append(labels, m.componentPreview(r.entry.Key, part))
	}
	start, end := m.switchWindow(labels, width, labelWidth, focus)
	position := labelWidth + 1
	for i := start; i < end; i++ {
		cellWidth := 4 + lipgloss.Width(labels[i])
		if x >= position && x < position+cellWidth {
			return i + 1, true
		}
		position += cellWidth + 1
	}
	return 0, false
}

func (m *model) mouseDialog(x, y, width, height int, mouse tea.Mouse, click bool) tea.Cmd {
	d := m.modal
	if d.kind == "select" {
		if !click {
			step := 1
			if mouse.Button == tea.MouseWheelUp {
				step = -1
			}
			d.selected = max(0, min(len(d.choices)-1, d.selected+step))
			return nil
		}
		listWidth := width
		if d.field == "animation" && width >= 64 {
			listWidth = min(30, width-34)
		}
		available := height - 7
		if x >= listWidth || y < 2 || y >= 2+available {
			return nil
		}
		start, _ := window(len(d.choices), d.selected, available)
		index := start + y - 2
		if index >= len(d.choices) {
			return nil
		}
		d.selected = index
		return m.updateDialog(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	}
	if d.kind == "confirm" && click && y == 2 {
		no := m.tr("Keep editing")
		yes := m.tr("Discard")
		if d.action != "cancel" {
			no = m.tr("Back")
			yes = m.tr("Confirm")
		}
		firstWidth := lipgloss.Width("[ " + no + " ]")
		if x < firstWidth {
			d.selected = 0
		} else if x >= firstWidth+2 && x < firstWidth+2+lipgloss.Width("[ "+yes+" ]") {
			d.selected = 1
		} else {
			return nil
		}
		return m.updateDialog(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	}
	return nil
}
