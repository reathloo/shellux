package settingsui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/reathloo/shellux/internal/config"
	"github.com/reathloo/shellux/internal/locale"
)

var sections = []string{"Menu items", "Menu layouts", "Themes", "Appearance", "General"}

func fit(s string, width int) string { return ansi.Truncate(s, max(0, width), "…") }
func line(s string, width int) string {
	s = fit(s, width)
	return s + strings.Repeat(" ", max(0, width-lipgloss.Width(s)))
}
func wrapped(s string, width, height int) string {
	lines := strings.Split(ansi.Wrap(s, max(1, width), ""), "\n")
	if len(lines) > height {
		lines = lines[:height]
		lines[height-1] = fit(lines[height-1], max(1, width-1)) + "…"
	}
	for len(lines) < height {
		lines = append(lines, "")
	}
	for i := range lines {
		lines[i] = line(lines[i], width)
	}
	return strings.Join(lines, "\n")
}
func window(length, selected, height int) (int, int) {
	start := max(0, selected-height+1)
	start = min(start, max(0, length-height))
	return start, min(length, start+height)
}
func (m *model) View() tea.View {
	width, height := max(1, m.width), max(1, m.height)
	styles := m.styles()
	var block string
	if m.small() {
		text := m.tr("Shellux · Settings\n\nResize the window to at least 64 × 18.\nYour changes are preserved.\nEsc: cancel · Ctrl+C: discard")
		if m.modal != nil && m.modal.kind == "confirm" && m.modal.action == "cancel" {
			a, b := m.tr("[Keep editing]"), m.tr("[Discard]")
			if m.modal.selected == 0 {
				a = styles.focused.Render(a)
			} else {
				b = styles.focused.Render(b)
			}
			text = m.tr("Unsaved changes\n\n") + a + "\n" + b + m.tr("\n←/→: select · Enter: confirm · Esc: back")
		}
		block = wrapped(text, max(1, min(60, width-2)), min(height, 7))
	} else {
		bw, bh := min(96, width-2), min(30, height-2)
		iw := bw - 4
		body := ""
		if m.modal != nil {
			body = m.dialogView(iw, bh-2, styles)
		} else {
			body = m.mainView(iw, bh-2, styles)
		}
		block = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(styles.border).Padding(0, 1).Render(body)
	}
	content := lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, block)
	v := tea.NewView(content)
	v.AltScreen = true
	v.WindowTitle = "Shellux"
	return v
}

func (m *model) mainView(width, height int, styles uiStyles) string {
	accent, muted, focused, normal := styles.accent, styles.muted, styles.focused, styles.normal
	bodyHeight := height - 9
	rw := width - 21
	nav := make([]string, bodyHeight)
	content := make([]string, bodyHeight)
	for i := range nav {
		nav[i] = line("", 18)
		content[i] = line("", rw)
	}
	for i, name := range sections {
		prefix := "  "
		if i == m.section {
			prefix = "› "
		}
		text := line(prefix+m.tr(name), 18)
		if i == m.section {
			if m.focus == 0 {
				text = focused.Render(text)
			} else {
				text = styles.heading(text, i*3)
			}
		} else {
			text = styles.secondary.Render(text)
		}
		nav[i] = text
	}
	if bodyHeight > 6 {
		nav[6] = muted.Render(line(fmt.Sprintf(m.tr("%d / 11 slots"), config.MenuSlots(m.draft.Visible)), 18))
	}
	if bodyHeight > 8 {
		state := m.tr("Saved")
		if m.dirty() {
			state = m.tr("Unsaved")
		}
		nav[8] = muted.Render(line(state, 18))
	}
	rows := m.rows()
	selected := min(m.cursors[m.section], len(rows)-1)
	start, end := window(len(rows), selected, bodyHeight)
	for i := start; i < end; i++ {
		r := rows[i]
		isFocus := m.focus == 1 && i == selected
		text := ""
		if r.heading {
			prefix := "  "
			if i == selected {
				prefix = "› "
			}
			text = styles.heading(prefix+"▾ "+m.tr(r.label), i*3)
			if isFocus {
				text = focused.Render(line(text, rw))
			}
		} else if r.action == "toggle" {
			text = m.switchRow(r, i == selected, isFocus, rw, styles)
		} else {
			prefix := "  "
			if i == selected {
				prefix = "› "
			}
			text = prefix + m.tr(r.label)
			if r.value != "" && r.action != "theme" && r.action != "preset" {
				value := r.value
				if r.action == "language" {
					value = m.tr(value)
				}
				text += "  · " + value
			}
			if r.color != "" && r.color != "default" {
				text += " " + lipgloss.NewStyle().Foreground(lipgloss.Color(r.color)).Render("██")
			}
			if isFocus {
				text = focused.Render(line(text, rw))
			} else {
				text = normal.Render(text)
			}
		}
		content[i-start] = line(text, rw)
	}
	body := make([]string, bodyHeight)
	for i := range body {
		body[i] = nav[i] + muted.Render(" │ ") + content[i]
	}
	title := m.tr("SHELLUX  ·  Settings")
	if bodyHeight <= 6 {
		title += fmt.Sprintf(m.tr("  ·  %d/11 slots"), config.MenuSlots(m.draft.Visible))
	}
	if start > 0 || end < len(rows) {
		title += fmt.Sprintf("  ·  %d–%d / %d", start+1, end, len(rows))
	}
	selectedRow := rows[selected]
	description := m.tr(selectedRow.description)
	if m.section == 0 && m.focus == 1 && selectedRow.action == "toggle" {
		part := ""
		if m.part >= 2 {
			part = selectedRow.entry.Parts[m.part-2]
		}
		description = m.menuHelp(selectedRow.entry.Key, part)
	}
	status := m.tr("Changes are applied only when you save.")
	if m.notice != "" {
		description = m.tr(m.notice)
		status = m.tr("Notice")
	}
	save, cancel := normal.Render(m.tr("[ Save ]")), normal.Render(m.tr("[ Cancel ]"))
	if m.focus == 2 {
		save = focused.Render(m.tr("[ Save ]"))
	}
	if m.focus == 3 {
		cancel = focused.Render(m.tr("[ Cancel ]"))
	}
	help := m.tr("Tab / Shift+Tab: focus · ↑/↓: select · Enter: open")
	if m.moveMode {
		help = m.tr("Enter/Esc: done · ↑/↓: item/section · ←/→: switch")
	} else if m.focus == 1 && m.section == 0 {
		switch {
		case selectedRow.action == "group":
			help = m.tr("M: move section · ↑/↓: select")
		case selectedRow.action == "toggle" && m.part == 0:
			help = m.tr("M: move item · →: switches · Space: toggle")
		case selectedRow.action == "toggle" && m.part == 1:
			help = m.tr("Space: toggle · ←/→: select · M: move item")
		case selectedRow.action == "toggle":
			help = m.tr("Shift+←/→: move switch · Space: toggle · M: move item")
		}
	}
	if m.focus == 1 && m.section == presetsSection {
		help = m.tr("↑/↓: layout · Enter: apply/save · Delete: remove saved layout")
	}
	if m.focus == 1 && m.section == themesSection {
		help = m.tr("↑/↓: theme · Enter: apply · F: favorite · U: uninstall")
	}
	return strings.Join([]string{
		line(styles.heading(title, 0), width), line("", width), strings.Join(body, "\n"), line("", width),
		muted.Render(wrapped(description, width, 2)), line(accent.Render(status), width), line(save+"  "+cancel, width),
		muted.Render(line(help, width)), muted.Render(line(m.tr("Ctrl+S: save · Esc: cancel · Ctrl+C: discard"), width)),
	}, "\n")
}

func (m *model) switchRow(r row, selectedRow, focus bool, width int, styles uiStyles) string {
	focused, normal := styles.focused, styles.normal
	prefix := "  "
	if selectedRow {
		prefix = "› "
	}
	label := prefix + fmt.Sprintf("%-8s", m.tr(r.label))
	text := ""
	if focus && m.part == 0 {
		text = focused.Underline(true).Render(label)
	} else {
		text = styles.secondary.Render(label)
	}
	masterLabel := m.tr("Enabled")
	if len(r.entry.Parts) == 0 {
		masterLabel += " · " + m.entryPreview(r.entry.Key)
	}
	labels := []string{masterLabel}
	keys := []string{r.entry.Key}
	for _, p := range r.entry.Parts {
		keys = append(keys, r.entry.Key+"."+p)
		labels = append(labels, m.componentPreview(r.entry.Key, p))
	}
	// Scroll controls horizontally in narrow windows, keeping the focused one
	// completely visible. Its semantic name remains in the help area.
	selected := 0
	if focus && m.part > 0 {
		selected = min(m.part-1, len(keys)-1)
	}
	start, end := selected, selected+1
	available := width - lipgloss.Width(text) - 2
	used := lipgloss.Width(labels[selected]) + 5
	for start > 0 && used+lipgloss.Width(labels[start-1])+5 <= available {
		start--
		used += lipgloss.Width(labels[start]) + 5
	}
	for end < len(keys) && used+lipgloss.Width(labels[end])+5 <= available {
		used += lipgloss.Width(labels[end]) + 5
		end++
	}
	if start > 0 {
		text += "‹"
	} else {
		text += " "
	}
	for i := start; i < end; i++ {
		key := keys[i]
		check := "[ ]"
		if config.Shown(m.draft.Visible, key) {
			check = "[✓]"
		}
		cell := check + " " + labels[i]
		if focus && m.part > 0 && i == m.part-1 {
			cell = focused.Underline(true).Render(cell)
		} else {
			cell = normal.Render(cell)
		}
		text += cell + " "
	}
	if end < len(keys) {
		text += "›"
	}
	return text
}

func (m *model) entryPreview(entry string) string {
	switch entry {
	case "platform":
		return "macOS · arm64 · Mac"
	case "shell":
		return "zsh · Terminal"
	case "uptime":
		return "2h 5m"
	case "date":
		return m.componentPreview(entry, "value")
	case "hostname":
		return "MacBook"
	case "network-status":
		return m.tr("Connected") + " · Wi-Fi"
	case "network-traffic":
		return "↑1.4 · ↓8.2 MB/s"
	case "ip":
		return "192.168.1.24"
	case "cpu":
		return "████░░░░ 42%"
	case "cores":
		return "8 " + m.tr("cores") + " · 42%"
	case "ram":
		return "██████░░ 6.8 GiB"
	case "volume":
		return "██████░░ 179/245G"
	case "battery":
		return "42% ⚡"
	case "temperature":
		return m.tr("Normal")
	case "directory":
		return "~/dev/shellux"
	case "spotify":
		return "Artist — Song"
	case "progress":
		return "▶ 1:24 ███░░░ 3:42"
	case "time":
		return "14:32:08"
	}
	return config.DisplayMenuItem(entry)
}

func (m *model) componentPreview(entry, part string) string {
	switch part {
	case "bar":
		return "████░░░░"
	case "sparkline":
		return "▂▃▅▇▆▄▂▃"
	case "percent":
		return "42%"
	case "top":
		return "code (24%)"
	case "pressure":
		return m.tr("Normal")
	case "remaining":
		return "~3h 20m"
	case "interface":
		return "en0"
	case "ip":
		return "192.168.1.24"
	case "connection":
		return "Wi-Fi"
	case "name":
		return "Home Wi-Fi"
	case "timezone":
		return "Berlin"
	case "utc":
		return "UTC+02:00"
	case "upload":
		return "↑ 1.4 MB/s"
	case "download":
		return "↓ 8.2 MB/s"
	case "latency":
		return "18 ms"
	case "value":
		switch entry {
		case "cpu":
			return "1.92 " + m.tr("load")
		case "ram":
			return "6.8 GiB"
		case "volume":
			return "179 / 245 GB"
		case "battery":
			return "⚡"
		case "cores":
			return "8 " + m.tr("cores")
		case "date":
			if m.language() == locale.German {
				return "15.10.2026"
			}
			return "2026-10-15"
		case "network-status":
			return m.tr("Connected")
		}
	}
	return m.tr(config.PartLabel(entry, part))
}

func (m *model) dialogView(width, height int, styles uiStyles) string {
	accent, muted, focused := styles.accent, styles.muted, styles.focused
	d := m.modal
	lines := []string{line(styles.heading(m.tr(d.title), 0), width), line("", width)}
	available := height - 7
	switch d.kind {
	case "input":
		lines = append(lines, line(d.input.View(), width))
		description := m.tr("Enter: apply · Esc: back")
		if d.field == "theme" || d.field == "preset" {
			description = m.tr("Name: a–z, 0–9 and hyphens (1–32 characters).")
		}
		lines = append(lines, muted.Render(line(description, width)))
	case "confirm":
		no, yes := m.tr("Keep editing"), m.tr("Discard")
		if d.action != "cancel" {
			no = m.tr("Back")
			yes = m.tr("Confirm")
		}
		a, b := "[ "+no+" ]", "[ "+yes+" ]"
		if d.selected == 0 {
			a = focused.Render(a)
		} else {
			b = focused.Render(b)
		}
		lines = append(lines, line(a+"  "+b, width))
		if d.action != "cancel" {
			lines = append(lines, muted.Render(line(m.tr("This change is applied permanently only when you save."), width)))
		}
	case "select":
		start, end := window(len(d.choices), d.selected, available)
		for i := start; i < end; i++ {
			option := d.choices[i]
			text := "  " + m.tr(option.label)
			if option.color != "" && option.color != "default" {
				text += " " + lipgloss.NewStyle().Foreground(lipgloss.Color(option.color)).Render("██")
			}
			if i == d.selected {
				text = "› " + strings.TrimPrefix(text, "  ")
				text = focused.Render(line(text, width))
			}
			lines = append(lines, line(text, width))
		}
		if start > 0 || end < len(d.choices) {
			lines = append(lines, muted.Render(line(fmt.Sprintf(m.tr("%d–%d / %d · ↑/↓ to scroll"), start+1, end, len(d.choices)), width)))
		}
	}
	for len(lines) < height-4 {
		lines = append(lines, line("", width))
	}
	lines = append(lines, accent.Render(wrapped(m.tr(d.error), width, 2)))
	help := m.tr("Enter: apply · Esc: back · Ctrl+C: discard")
	if d.kind == "select" {
		help = m.tr("↑/↓: select · Enter: apply · Esc: back")
	}
	if d.kind == "confirm" {
		help = m.tr("←/→ or Tab: select · Enter: confirm · Esc: back")
	}
	lines = append(lines, muted.Render(line(help, width)), muted.Render(line(m.tr("The file stays unchanged until you save."), width)))
	return strings.Join(lines, "\n")
}
