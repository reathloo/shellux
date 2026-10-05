package settingsui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/reathloo/shellux/internal/config"
)

var sections = []string{"Menu items", "Themes", "Appearance", "General"}

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
		text := "Shellux · Settings\n\nResize the window to at least 64 × 18.\nYour changes are preserved.\nEsc: cancel · Ctrl+C: discard"
		if m.modal != nil && m.modal.kind == "confirm" && m.modal.action == "cancel" {
			a, b := "[Keep editing]", "[Discard]"
			if m.modal.selected == 0 {
				a = styles.focused.Render(a)
			} else {
				b = styles.focused.Render(b)
			}
			text = "Unsaved changes\n\n" + a + "\n" + b + "\n←/→: select · Enter: confirm · Esc: back"
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
		text := line(prefix+name, 18)
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
		nav[6] = muted.Render(line(fmt.Sprintf("%d / 11 slots", config.MenuSlots(m.draft.Visible)), 18))
	}
	if bodyHeight > 8 {
		state := "Saved"
		if m.dirty() {
			state = "Unsaved"
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
			text = styles.heading(r.label, i*3)
		} else if r.action == "toggle" {
			text = m.switchRow(r, i == selected, isFocus, rw, styles)
		} else {
			prefix := "  "
			if i == selected {
				prefix = "› "
			}
			text = prefix + r.label
			if r.value != "" && r.action != "theme" {
				text += "  · " + r.value
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
	title := "SHELLUX  ·  Settings"
	if bodyHeight <= 6 {
		title += fmt.Sprintf("  ·  %d/11 slots", config.MenuSlots(m.draft.Visible))
	}
	if start > 0 || end < len(rows) {
		title += fmt.Sprintf("  ·  %d–%d / %d", start+1, end, len(rows))
	}
	selectedRow := rows[selected]
	description := selectedRow.description
	if m.section == 0 && m.focus == 1 && len(selectedRow.entry.Parts) > 0 {
		label := "Enabled"
		if m.part > 0 {
			label = config.PartLabel(selectedRow.entry.Key, selectedRow.entry.Parts[m.part-1])
		}
		description = "Selected: " + label + ". " + description
	}
	status := "Changes are applied only when you save."
	if m.notice != "" {
		description = m.notice
		status = "Notice"
	}
	save, cancel := normal.Render("[ Save ]"), normal.Render("[ Cancel ]")
	if m.focus == 2 {
		save = focused.Render("[ Save ]")
	}
	if m.focus == 3 {
		cancel = focused.Render("[ Cancel ]")
	}
	help := "Tab / Shift+Tab: focus · ↑/↓: select · Enter: open"
	if m.focus == 1 && m.section == 0 {
		help = "←/→: switch · Space: toggle · Tab: focus"
	}
	if m.focus == 1 && m.section == 1 {
		help = "↑/↓: theme · Enter: apply · Delete: remove custom theme"
	}
	return strings.Join([]string{
		line(styles.heading(title, 0), width), line("", width), strings.Join(body, "\n"), line("", width),
		muted.Render(wrapped(description, width, 2)), line(accent.Render(status), width), line(save+"  "+cancel, width),
		muted.Render(line(help, width)), muted.Render(line("Ctrl+S: save · Esc: cancel · Ctrl+C: discard", width)),
	}, "\n")
}

func (m *model) switchRow(r row, selectedRow, focus bool, width int, styles uiStyles) string {
	focused, normal := styles.focused, styles.normal
	prefix := "  "
	if selectedRow {
		prefix = "› "
	}
	text := styles.secondary.Render(prefix + fmt.Sprintf("%-8s", r.label))
	labels := []string{"Enabled"}
	keys := []string{r.entry.Key}
	for _, p := range r.entry.Parts {
		keys = append(keys, r.entry.Key+"."+p)
		labels = append(labels, config.PartLabel(r.entry.Key, p))
	}
	length := lipgloss.Width(text)
	for _, label := range labels {
		length += len(label) + 5
	}
	if length > width {
		labels[0] = "On"
		compact := map[string]string{"bar": "B", "percent": "%", "value": "V", "top": "Top", "pressure": "P", "remaining": "Time", "interface": "If", "ip": "IP", "connection": "Conn", "name": "Name", "timezone": "TZ", "utc": "UTC", "upload": "Up", "download": "Down", "latency": "Ping"}
		for i, part := range r.entry.Parts {
			labels[i+1] = compact[part]
		}
	}
	// Scroll controls horizontally in narrow windows, keeping the focused one
	// completely visible. Its unabbreviated label is always in the help area.
	selected := 0
	if focus {
		selected = min(m.part, len(keys)-1)
	}
	start, end := selected, selected+1
	available := width - lipgloss.Width(text) - 2
	used := len(labels[selected]) + 5
	for start > 0 && used+len(labels[start-1])+5 <= available {
		start--
		used += len(labels[start]) + 5
	}
	for end < len(keys) && used+len(labels[end])+5 <= available {
		used += len(labels[end]) + 5
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
			check = "[x]"
		}
		cell := check + " " + labels[i]
		if focus && i == m.part {
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

func (m *model) dialogView(width, height int, styles uiStyles) string {
	accent, muted, focused := styles.accent, styles.muted, styles.focused
	d := m.modal
	lines := []string{line(styles.heading(d.title, 0), width), line("", width)}
	available := height - 7
	switch d.kind {
	case "input":
		lines = append(lines, line(d.input.View(), width))
		description := "Enter: apply · Esc: back"
		if d.field == "theme" {
			description = "Name: a–z, 0–9 and hyphens (1–32 characters)."
		}
		lines = append(lines, muted.Render(line(description, width)))
	case "confirm":
		no, yes := "Keep editing", "Discard"
		if d.action != "cancel" {
			no = "Back"
			yes = "Confirm"
		}
		a, b := "[ "+no+" ]", "[ "+yes+" ]"
		if d.selected == 0 {
			a = focused.Render(a)
		} else {
			b = focused.Render(b)
		}
		lines = append(lines, line(a+"  "+b, width))
		if d.action != "cancel" {
			lines = append(lines, muted.Render(line("This change is applied permanently only when you save.", width)))
		}
	case "select":
		start, end := window(len(d.choices), d.selected, available)
		for i := start; i < end; i++ {
			option := d.choices[i]
			text := "  " + option.label
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
			lines = append(lines, muted.Render(line(fmt.Sprintf("%d–%d / %d · ↑/↓ to scroll", start+1, end, len(d.choices)), width)))
		}
	}
	for len(lines) < height-4 {
		lines = append(lines, line("", width))
	}
	lines = append(lines, accent.Render(wrapped(d.error, width, 2)))
	help := "Enter: apply · Esc: back · Ctrl+C: discard"
	if d.kind == "select" {
		help = "↑/↓: select · Enter: apply · Esc: back"
	}
	if d.kind == "confirm" {
		help = "←/→ or Tab: select · Enter: confirm · Esc: back"
	}
	lines = append(lines, muted.Render(line(help, width)), muted.Render(line("The file stays unchanged until you save.", width)))
	return strings.Join(lines, "\n")
}
