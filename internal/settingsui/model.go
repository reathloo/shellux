// Package settingsui edits a private configuration draft on an alternate screen.
package settingsui

import (
	"context"
	"errors"
	"fmt"
	"os"
	"reflect"
	"slices"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/term"
	"github.com/reathloo/shellux/internal/config"
	"github.com/reathloo/shellux/internal/themepack"
)

const (
	Saved       = 0
	Cancelled   = 1
	Failed      = 2
	Interrupted = 130
)

const (
	menuSection = iota
	presetsSection
	themesSection
	appearanceSection
	generalSection
	sectionCount
)

// Run always lets Bubble Tea restore the terminal before returning an exit code.
func Run() (int, config.Config, error) {
	if !term.IsTerminal(os.Stdin.Fd()) || !term.IsTerminal(os.Stdout.Fd()) {
		return Failed, config.Config{}, errors.New("shellux settings requires interactive standard input and output; open it in a terminal without redirection")
	}
	session, err := config.OpenEditor()
	if err != nil {
		return Failed, config.Config{}, err
	}
	m := newModel(session.Config, session.Save)
	m.ensure = func(ctx context.Context, c config.Config) error { return c.EnsurePackages(ctx) }
	m.refresh = true
	defer func() {
		if m.cancelDownload != nil {
			m.cancelDownload()
		}
		if m.cancelCatalog != nil {
			m.cancelCatalog()
		}
	}()
	final, err := tea.NewProgram(m, tea.WithInput(os.Stdin), tea.WithOutput(os.Stdout)).Run()
	if errors.Is(err, tea.ErrInterrupted) {
		return Interrupted, config.Config{}, nil
	}
	if err != nil {
		return Failed, config.Config{}, err
	}
	result := final.(*model)
	return result.result, result.draft, nil
}

type row struct {
	label, action, value, description string
	color                             string
	entry                             config.MenuEntry
	heading                           bool
	custom                            bool
}
type choice struct{ label, value, color string }
type dialog struct {
	kind, title, field, action, value string
	choices                           []choice
	selected                          int
	input                             textinput.Model
	error                             string
}
type model struct {
	original, draft         config.Config
	save                    func(config.Config) error
	width, height           int
	section, focus, part    int // focus: navigation, content, save, cancel
	cursors                 [sectionCount]int
	modal                   *dialog
	notice                  string
	moveMode                bool
	result                  int
	done                    bool
	ensure                  func(context.Context, config.Config) error
	cancelDownload          context.CancelFunc
	cancelCatalog           context.CancelFunc
	downloading             bool
	downloadID              int
	refresh, refreshStarted bool
	installedThemes         map[string]bool
	checkInstalled          func(string) bool
}

func newModel(c config.Config, save func(config.Config) error) *model {
	m := &model{original: c.Clone(), draft: c.Clone(), save: save, width: 96, height: 30, result: Cancelled, checkInstalled: themepack.IsInstalled}
	m.cursors[0] = 1 // first row is the SYSTEM heading
	return m
}
func (m *model) Init() tea.Cmd           { return nil }
func (m *model) dirty() bool             { return !reflect.DeepEqual(m.original, m.draft) }
func (m *model) small() bool             { return m.width < 64 || m.height < 18 }
func (m *model) finish(code int) tea.Cmd { m.result = code; m.done = true; return tea.Quit }
func (m *model) isInstalled(name string) bool {
	if installed, ok := m.installedThemes[name]; ok {
		return installed
	}
	if m.installedThemes == nil {
		m.installedThemes = make(map[string]bool)
	}
	installed := m.checkInstalled(name)
	m.installedThemes[name] = installed
	return installed
}
func (m *model) saveDraft() tea.Cmd {
	if err := m.draft.Validate(); err != nil {
		m.notice = friendly(err)
		return nil
	}
	if m.ensure != nil && len(m.draft.RequiredPackages()) > 0 {
		if m.downloading {
			return nil
		}
		ctx, cancel := context.WithCancel(context.Background())
		m.cancelDownload = cancel
		m.downloading = true
		m.downloadID++
		id := m.downloadID
		draft := m.draft.Clone()
		ensure := m.ensure
		m.notice = "Preparing selected themes… Esc: stop download"
		return func() tea.Msg { return downloadResult{id: id, err: ensure(ctx, draft)} }
	}
	return m.commitDraft()
}
func (m *model) commitDraft() tea.Cmd {
	if err := m.save(m.draft.Clone()); err != nil {
		m.notice = m.tr("Could not save: ") + m.tr(friendly(err))
		return nil
	}
	return m.finish(Saved)
}
func (m *model) cancel() tea.Cmd {
	if !m.dirty() {
		return m.finish(Cancelled)
	}
	m.modal = &dialog{kind: "confirm", title: "Discard unsaved changes?", action: "cancel"}
	return nil
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case downloadResult:
		if msg.id != m.downloadID || !m.downloading {
			return m, nil
		}
		m.downloading = false
		m.cancelDownload()
		m.cancelDownload = nil
		if msg.err != nil {
			m.notice = m.tr("Could not download: ") + m.tr(friendly(msg.err))
			return m, nil
		}
		return m, m.commitDraft()
	case catalogResult:
		if msg.err != nil {
			if m.downloading {
				return m, nil
			}
			m.notice = m.tr("Theme update check failed: ") + friendly(msg.err)
			return m, nil
		}
		selected := m.selectedRow().value
		themepack.UseCatalog(msg.catalog)
		if msg.updated > 0 {
			m.installedThemes = nil
			m.notice = fmt.Sprintf(m.tr("Updated %d installed theme(s)."), msg.updated)
		}
		for i, r := range m.rows() {
			if r.value == selected {
				m.cursors[m.section] = i
				break
			}
		}
		return m, nil
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		if m.modal != nil && m.modal.kind == "input" {
			m.modal.input.SetWidth(max(8, min(52, m.width-12)))
		}
		return m, nil
	case tea.KeyPressMsg:
		key := msg.String()
		if m.downloading {
			if key == "ctrl+c" {
				m.cancelDownload()
				m.downloading = false
				m.downloadID++
				return m, m.finish(Interrupted)
			}
			if key == "esc" {
				m.cancelDownload()
				m.cancelDownload = nil
				m.downloading = false
				m.downloadID++
				m.notice = "Download cancelled. Your changes are preserved."
			}
			return m, nil
		}
		if key == "ctrl+c" {
			return m, m.finish(Interrupted)
		}
		if m.small() {
			if key == "esc" {
				if m.modal != nil {
					m.modal = nil
					return m, nil
				}
				return m, m.cancel()
			}
			// Keep cancellation operable even if the confirmation cannot fit normally.
			if m.modal != nil && m.modal.kind == "confirm" && m.modal.action == "cancel" {
				return m, m.updateDialog(msg)
			}
			return m, nil
		}
		if m.modal != nil {
			return m, m.updateDialog(msg)
		}
		if m.moveMode {
			switch key {
			case "m", "enter", "esc":
				m.moveMode = false
			case "up", "down":
				step := 1
				if key == "up" {
					step = -1
				}
				m.moveMenuRow(step)
			case "left", "right":
				step := 1
				if key == "left" {
					step = -1
				}
				m.moveMenuPart(step)
			case "ctrl+s":
				return m, m.saveDraft()
			}
			return m, nil
		}
		switch key {
		case "ctrl+s":
			return m, m.saveDraft()
		case "esc":
			return m, m.cancel()
		case "tab":
			m.focus = (m.focus + 1) % 4
		case "shift+tab":
			m.focus = (m.focus + 3) % 4
		case "m":
			if m.focus == 1 && m.section == 0 {
				r := m.selectedRow()
				m.moveMode = r.action == "group" || r.action == "toggle"
			}
		case "shift+up", "shift+down":
			if m.focus == 1 && m.section == 0 {
				step := 1
				if key == "shift+up" {
					step = -1
				}
				m.moveMenuRow(step)
			}
		case "shift+left", "shift+right":
			if m.focus == 1 && m.section == 0 {
				step := 1
				if key == "shift+left" {
					step = -1
				}
				m.moveMenuPart(step)
			}
		case "up", "down":
			step := 1
			if key == "up" {
				step = -1
			}
			if m.focus == 0 {
				m.section = (m.section + step + sectionCount) % sectionCount
				m.part = 0
				if m.section == themesSection && m.refresh && !m.refreshStarted {
					return m, m.refreshCatalog()
				}
			} else if m.focus == 1 {
				m.moveRow(step)
			}
		case "left", "right":
			if m.focus == 1 && m.section == 0 {
				r := m.selectedRow()
				n := len(r.entry.Parts) + 2 // item name, Enabled, components
				if r.action == "toggle" {
					step := 1
					if key == "left" {
						step = -1
					}
					m.part = (m.part + step + n) % n
				}
			} else if m.focus >= 2 {
				if m.focus == 2 {
					m.focus = 3
				} else {
					m.focus = 2
				}
			}
		case "delete", "backspace":
			if m.focus == 1 && m.section == themesSection {
				r := m.selectedRow()
				if r.custom {
					m.modal = &dialog{kind: "confirm", title: fmt.Sprintf(m.tr("Delete custom theme %q?"), r.value), action: "delete", value: r.value}
				} else {
					m.notice = "Catalog themes are protected."
				}
			} else if m.focus == 1 && m.section == presetsSection {
				r := m.selectedRow()
				if r.action == "preset" && r.custom {
					m.modal = &dialog{kind: "confirm", title: fmt.Sprintf(m.tr("Delete menu preset %q?"), r.value), action: "delete-preset", value: r.value}
				}
			}
		case "f":
			if m.focus == 1 && m.section == themesSection {
				r := m.selectedRow()
				if r.action == "theme" {
					if err := m.draft.ToggleFavoriteTheme(r.value); err != nil {
						m.notice = friendly(err)
					} else {
						m.selectThemeRow(r.value)
					}
				}
			}
		case "u":
			if m.focus == 1 && m.section == themesSection {
				r := m.selectedRow()
				if r.action == "theme" && m.isInstalled(r.value) {
					if slices.Contains(m.original.RequiredPackages(), r.value) || slices.Contains(m.draft.RequiredPackages(), r.value) {
						m.notice = m.tr("Select and save another theme before uninstalling this one.")
					} else {
						m.modal = &dialog{kind: "confirm", title: fmt.Sprintf(m.tr("Uninstall downloaded theme %q now?"), r.value), action: "uninstall", value: r.value}
					}
				}
			}
		case "space", "enter":
			switch m.focus {
			case 0:
				m.focus = 1
			case 1:
				return m, m.activate()
			case 2:
				return m, m.saveDraft()
			case 3:
				return m, m.cancel()
			}
		}
	case tea.PasteMsg:
		if !m.small() && m.modal != nil && m.modal.kind == "input" {
			return m, m.updateDialog(msg)
		}
	}
	if !m.small() && m.modal != nil && m.modal.kind == "input" {
		return m, m.updateDialog(msg)
	}
	return m, nil
}

func (m *model) rows() []row {
	switch m.section {
	case 0:
		rows := []row{}
		group := ""
		for _, e := range m.draft.OrderedMenuEntries() {
			if e.Group != group {
				group = e.Group
				rows = append(rows, row{label: group, action: "group", value: group, description: "Press M, then use Up/Down to move this complete section.", heading: true})
			}
			rows = append(rows, row{label: e.Name, action: "toggle", entry: e})
		}
		return append(rows, row{label: "Reset menu items", action: "reset", description: "Reset items, components and their order; appearance stays unchanged."})
	case presetsSection:
		rows := []row{}
		for _, name := range m.draft.MenuPresetOptions() {
			mark := "[ ]"
			if m.draft.MenuPresetMatches(name) {
				mark = "[✓]"
			}
			description := "Apply this menu layout: visible items, detail switches and order. Theme and general settings stay unchanged."
			if name == "default" {
				description = "Restore the built-in menu layout. Saved layouts stay available."
			} else {
				description += " Delete removes only this saved layout."
			}
			label := name
			if name == "default" {
				label = m.tr("Default menu")
			}
			rows = append(rows, row{label: mark + " " + label, action: "preset", value: name, custom: name != "default", description: description})
		}
		return append(rows, row{label: "Save current menu as a layout…", action: "new-preset", description: "Save visible menu items, detail switches and order under a name; theme and general settings stay unchanged."})
	case themesSection:
		rows := []row{}
		for _, t := range m.draft.ThemeOptions() {
			desc := fmt.Sprintf(m.tr("Animation: %s · Style: %s · Background: %s"), t.Animation, t.Style, config.DisplayBackground(t.Background))
			if t.Custom {
				desc += m.tr(" · Delete: remove custom theme")
			}
			if t.Name != "default" {
				desc += m.tr(" · F: favorite")
			}
			mark := "[ ]"
			if m.draft.Animation == t.Animation && config.DisplayStyle(m.draft.Style) == t.Style && m.draft.Background == t.Background {
				mark = "[✓]"
			}
			kind := m.tr("built-in")
			if e, ok := themepack.Lookup(t.Name); ok {
				kind = fmt.Sprintf("Download · %.0f KB", float64(e.Size)/1024)
				if m.isInstalled(t.Name) {
					kind = m.tr("Installed")
					desc += m.tr(" · U: uninstall now (keeps theme in catalog)")
				} else {
					desc += m.tr(" · Downloaded only when you save.")
				}
			}
			if t.Custom {
				kind = m.tr("custom")
			}
			favorite := ""
			if m.draft.FavoriteThemes[t.Name] {
				favorite = "★ "
			}
			rows = append(rows, row{label: mark + " " + favorite + t.Name + " · " + kind, action: "theme", value: t.Name, color: t.Background, custom: t.Custom, description: desc})
		}
		rows = append(rows, row{label: "Random combination", action: "random", description: "Combine an animation with a random style and matching background."}, row{label: "Save current combination as a theme…", action: "new", description: fmt.Sprintf(m.tr("Current: %s · %s · %s. Save as a custom theme."), m.draft.Animation, config.DisplayStyle(m.draft.Style), m.draft.BackgroundName())})
		return rows
	case appearanceSection:
		return []row{
			{label: "Animation", action: "animation", value: m.draft.Animation, description: "Choose an animation; style and background stay unchanged."},
			{label: "Style", action: "style", value: config.DisplayStyle(m.draft.Style), description: "Choose header colors independently of the theme."},
			{label: "Background", action: "background", value: m.draft.BackgroundName(), color: m.draft.Background, description: "Named colors, theme backgrounds, profile color or a custom RGB color."},
		}
	default:
		return []row{{label: "Refresh interval", action: "interval", value: m.draft.RefreshInterval, description: "A positive Go duration, such as 1s, 500ms or 2.5s."}, {label: "Language", action: "language", value: languageName(m.draft.Language), description: "Choose the dashboard and settings language. Automatic follows your system locale."}}
	}
}

func (m *model) selectThemeRow(name string) {
	for i, r := range m.rows() {
		if r.action == "theme" && r.value == name {
			m.cursors[themesSection] = i
			return
		}
	}
}
func (m *model) selectedRow() row {
	rows := m.rows()
	i := min(m.cursors[m.section], len(rows)-1)
	return rows[max(0, i)]
}
func (m *model) moveRow(step int) {
	rows := m.rows()
	next := m.cursors[m.section] + step
	if next >= 0 && next < len(rows) {
		m.cursors[m.section] = next
		m.part = 0
	}
}

func (m *model) selectMenuRow(action, value string) {
	for index, candidate := range m.rows() {
		if candidate.action == action && (candidate.value == value || candidate.entry.Key == value) {
			m.cursors[0] = index
			return
		}
	}
}

func (m *model) moveMenuRow(step int) {
	r := m.selectedRow()
	moved := false
	switch {
	case r.action == "group":
		moved = m.draft.MoveMenuGroup(r.value, step)
		m.selectMenuRow("group", r.value)
	case r.action == "toggle":
		moved = m.draft.MoveMenuItem(r.entry.Key, step)
		m.selectMenuRow("toggle", r.entry.Key)
	default:
		return
	}
	if moved {
		m.notice = "Menu order changed. Save to apply it."
	} else if r.action == "group" && r.value == "HEADER" {
		m.notice = "The header clock is fixed in the title."
	} else {
		m.notice = "Already at the edge of this section."
	}
}

func (m *model) moveMenuPart(step int) {
	r := m.selectedRow()
	if r.action != "toggle" || m.part < 2 || m.part > len(r.entry.Parts)+1 {
		return
	}
	part := r.entry.Parts[m.part-2]
	if m.draft.MoveDisplayPart(r.entry.Key, part, step) {
		m.part += step
		m.notice = "Component order changed. Save to apply it."
	} else {
		m.notice = "Already at the edge of this item."
	}
}
func (m *model) activate() tea.Cmd {
	m.notice = ""
	r := m.selectedRow()
	switch r.action {
	case "toggle":
		p := ""
		if m.part >= 2 {
			p = r.entry.Parts[m.part-2]
		}
		key := r.entry.Key
		if p != "" {
			key += "." + p
		}
		if err := m.draft.ChangeDisplay(r.entry.Key, p, !config.Shown(m.draft.Visible, key)); err != nil {
			m.notice = friendly(err)
		}
	case "reset":
		m.draft.ResetMenu()
		m.notice = "Menu items and components reset."
	case "preset":
		if err := m.draft.ApplyMenuPreset(r.value); err != nil {
			m.notice = friendly(err)
		}
	case "new-preset":
		return m.openInput("Save menu layout", "preset", "", "Name: a–z, 0–9 and hyphens, up to 32 characters")
	case "theme":
		_, err := m.draft.ApplyTheme(r.value)
		if err != nil {
			m.notice = friendly(err)
		}
	case "random":
		_, err := m.draft.ApplyTheme("random")
		if err != nil {
			m.notice = friendly(err)
		} else {
			m.notice = fmt.Sprintf(m.tr("Random: %s · %s · %s"), m.draft.Animation, config.DisplayStyle(m.draft.Style), m.draft.BackgroundName())
		}
	case "new":
		return m.openInput("Save custom theme", "theme", "", "Name: a–z, 0–9 and hyphens, up to 32 characters")
	case "interval":
		return m.openInput("Refresh interval", "interval", m.draft.RefreshInterval, "1s or 500ms")
	case "animation", "style", "background", "language":
		options := []choice{}
		current := ""
		switch r.action {
		case "language":
			current = m.draft.Language
			options = []choice{{label: "Automatic (system)", value: "auto"}, {label: "English", value: "en"}, {label: "Deutsch", value: "de"}}
		case "animation":
			current = m.draft.Animation
			for _, v := range config.Animations() {
				options = append(options, choice{label: m.packageLabel(v), value: v})
			}
		case "style":
			current = config.DisplayStyle(m.draft.Style)
			for _, v := range config.Styles() {
				options = append(options, choice{label: m.packageLabel(v), value: v})
			}
		case "background":
			current = m.draft.Background
			for _, v := range m.draft.BackgroundOptions() {
				label := v.Name
				if strings.HasPrefix(label, "Theme: ") {
					label = "Theme: " + m.packageLabel(strings.TrimPrefix(label, "Theme: "))
				}
				options = append(options, choice{label: label, value: v.Color, color: v.Color})
			}
			options = append(options, choice{label: "Custom RGB color…", value: "custom"})
		}
		selected := 0
		for i, v := range options {
			if v.value == current {
				selected = i
				break
			}
		}
		m.modal = &dialog{kind: "select", title: "Select " + strings.ToLower(r.label), field: r.action, choices: options, selected: selected}
	}
	return nil
}

func (m *model) openInput(title, field, value, placeholder string) tea.Cmd {
	input := textinput.New()
	input.Prompt = "> "
	input.Placeholder = m.tr(placeholder)
	input.CharLimit = 64
	input.SetWidth(min(52, m.width-12))
	input.SetVirtualCursor(true)
	input.SetStyles(m.styles().input())
	input.SetValue(value)
	m.modal = &dialog{kind: "input", title: title, field: field, input: input}
	return m.modal.input.Focus()
}
func (m *model) updateDialog(msg tea.Msg) tea.Cmd {
	d := m.modal
	key := ""
	if k, ok := msg.(tea.KeyPressMsg); ok {
		key = k.String()
	}
	if key == "esc" {
		m.modal = nil
		return nil
	}
	switch d.kind {
	case "input":
		if key == "enter" || key == "ctrl+s" {
			if m.confirmInput() && key == "ctrl+s" && m.modal == nil {
				return m.saveDraft()
			}
			return nil
		}
		var cmd tea.Cmd
		d.input, cmd = d.input.Update(msg)
		return cmd
	case "select":
		switch key {
		case "up":
			d.selected = max(0, d.selected-1)
		case "down":
			d.selected = min(len(d.choices)-1, d.selected+1)
		case "enter", "space":
			value := d.choices[d.selected].value
			if d.field == "background" && value == "custom" {
				return m.openInput("Custom RGB color", "background", "#", "#RRGGBB")
			}
			if err := m.draft.ChangeAppearance(d.field, value); err != nil {
				d.error = friendly(err)
			} else {
				m.modal = nil
			}
		}
	case "confirm":
		switch key {
		case "left", "right", "up", "down", "tab", "shift+tab":
			d.selected = 1 - d.selected
		case "enter", "space":
			if d.selected == 0 {
				m.modal = nil
				return nil
			}
			switch d.action {
			case "cancel":
				return m.finish(Cancelled)
			case "delete":
				_, err := m.draft.RemoveTheme(d.value)
				if err != nil {
					m.notice = friendly(err)
				}
				m.cursors[themesSection] = min(m.cursors[themesSection], len(m.rows())-1)
			case "uninstall":
				store, err := themepack.DefaultStore()
				if err == nil {
					err = store.Uninstall(d.value)
				}
				if err != nil {
					m.notice = m.tr("Could not uninstall theme: ") + friendly(err)
				} else {
					m.installedThemes[d.value] = false
					m.notice = fmt.Sprintf(m.tr("Theme %q uninstalled."), d.value)
				}
			case "delete-preset":
				if err := m.draft.RemoveMenuPreset(d.value); err != nil {
					m.notice = friendly(err)
				}
				m.cursors[presetsSection] = min(m.cursors[presetsSection], len(m.rows())-1)
			case "overwrite":
				_, err := m.draft.StoreTheme(d.value)
				if err != nil {
					m.notice = friendly(err)
				}
			case "overwrite-preset":
				if _, err := m.draft.StoreMenuPreset(d.value); err != nil {
					m.notice = friendly(err)
				}
			}
			m.modal = nil
		}
	}
	return nil
}
func (m *model) confirmInput() bool {
	d := m.modal
	value := strings.TrimSpace(d.input.Value())
	if d.field == "theme" {
		// Validate on a temporary copy before asking about an existing name.
		test := m.draft.Clone()
		name, err := test.StoreTheme(value)
		if err != nil {
			d.error = friendly(err)
			return false
		}
		if _, exists := m.draft.Themes[name]; exists {
			m.modal = &dialog{kind: "confirm", title: fmt.Sprintf(m.tr("Overwrite custom theme %q?"), name), action: "overwrite", value: name}
			return false
		}
		m.draft = test
	} else if d.field == "preset" {
		test := m.draft.Clone()
		name, err := test.StoreMenuPreset(value)
		if err != nil {
			d.error = friendly(err)
			return false
		}
		if _, exists := m.draft.MenuPresets[name]; exists {
			m.modal = &dialog{kind: "confirm", title: fmt.Sprintf(m.tr("Overwrite menu preset %q?"), name), action: "overwrite-preset", value: name}
			return false
		}
		m.draft = test
	} else if err := m.draft.ChangeAppearance(d.field, value); err != nil {
		d.error = friendly(err)
		return false
	}
	m.modal = nil
	return true
}
func friendly(err error) string {
	if errors.Is(err, config.ErrChanged) {
		return "File changed externally. Please cancel and reopen settings."
	}
	if strings.Contains(err.Error(), "at most 11 slots") {
		return "All 11 slots are taken. Hide an item first."
	}
	if strings.Contains(err.Error(), "refresh interval") {
		return "Enter a positive duration, such as 1s or 500ms."
	}
	if strings.Contains(err.Error(), "invalid background") {
		return "Enter an RGB color in #RRGGBB format."
	}
	if strings.Contains(err.Error(), "theme name") {
		if strings.Contains(err.Error(), "reserved") || strings.Contains(err.Error(), "built in") {
			return "This theme name is protected. Choose a custom name."
		}
		return "Theme names: 1–32 letters (a–z), digits or hyphens; no leading or trailing hyphen."
	}
	if strings.Contains(err.Error(), "menu preset name") {
		return "Layout names: 1–32 letters (a–z), digits or hyphens; no leading or trailing hyphen."
	}
	return err.Error()
}

type downloadResult struct {
	id  int
	err error
}
type catalogResult struct {
	catalog themepack.Catalog
	updated int
	err     error
}

func (m *model) refreshCatalog() tea.Cmd {
	m.refreshStarted = true
	ctx, cancel := context.WithCancel(context.Background())
	m.cancelCatalog = cancel
	return func() tea.Msg {
		s, err := themepack.DefaultStore()
		if err != nil {
			return catalogResult{err: err}
		}
		c, err := s.Refresh(ctx)
		if err != nil {
			return catalogResult{err: err}
		}
		updated, err := s.UpdateInstalled(ctx, c)
		return catalogResult{catalog: c, updated: updated, err: err}
	}
}

func (m *model) packageLabel(name string) string {
	if e, ok := themepack.Lookup(name); ok {
		if m.isInstalled(name) {
			return name + " · " + m.tr("Installed")
		}
		return fmt.Sprintf("%s · Download %.0f KB", name, float64(e.Size)/1024)
	}
	return name
}
