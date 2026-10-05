// Package settingsui edits a private configuration draft on an alternate screen.
package settingsui

import (
	"context"
	"errors"
	"fmt"
	"os"
	"reflect"
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
	cursors                 [4]int
	modal                   *dialog
	notice                  string
	result                  int
	done                    bool
	ensure                  func(context.Context, config.Config) error
	cancelDownload          context.CancelFunc
	cancelCatalog           context.CancelFunc
	downloading             bool
	downloadID              int
	refresh, refreshStarted bool
}

func newModel(c config.Config, save func(config.Config) error) *model {
	m := &model{original: c.Clone(), draft: c.Clone(), save: save, width: 96, height: 30, result: Cancelled}
	m.cursors[0] = 1 // first row is the SYSTEM heading
	return m
}
func (m *model) Init() tea.Cmd           { return nil }
func (m *model) dirty() bool             { return !reflect.DeepEqual(m.original, m.draft) }
func (m *model) small() bool             { return m.width < 64 || m.height < 18 }
func (m *model) finish(code int) tea.Cmd { m.result = code; m.done = true; return tea.Quit }
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
			m.notice = "Theme catalog offline; showing saved choices."
			return m, nil
		}
		selected := m.selectedRow().value
		themepack.UseCatalog(msg.catalog)
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
		switch key {
		case "ctrl+s":
			return m, m.saveDraft()
		case "esc":
			return m, m.cancel()
		case "tab":
			m.focus = (m.focus + 1) % 4
		case "shift+tab":
			m.focus = (m.focus + 3) % 4
		case "up", "down":
			step := 1
			if key == "up" {
				step = -1
			}
			if m.focus == 0 {
				m.section = (m.section + step + 4) % 4
				m.part = 0
				if m.section == 1 && m.refresh && !m.refreshStarted {
					return m, m.refreshCatalog()
				}
			} else if m.focus == 1 {
				m.moveRow(step)
			}
		case "left", "right":
			if m.focus == 1 && m.section == 0 {
				r := m.selectedRow()
				n := len(r.entry.Parts) + 1
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
			if m.focus == 1 && m.section == 1 {
				r := m.selectedRow()
				if r.custom {
					m.modal = &dialog{kind: "confirm", title: fmt.Sprintf(m.tr("Delete custom theme %q?"), r.value), action: "delete", value: r.value}
				} else {
					m.notice = "Catalog themes are protected."
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
		for _, e := range config.MenuEntries() {
			if e.Group != group {
				group = e.Group
				rows = append(rows, row{label: group, heading: true})
			}
			description := "Show or hide an item; its component choices are preserved."
			switch e.Key {
			case "temperature":
				description = "Battery and thermal share one slot."
			case "battery":
				description = "Shares a slot with thermal. Remaining: estimated battery runtime; N/A while charging."
			case "spotify", "progress":
				description = "Spotify and progress share one slot."
			case "time":
				description = "The clock in the header does not use a menu slot."
			case "date":
				description = "Date, timezone name and UTC offset are independent. The offset follows daylight saving time."
			case "cpu":
				description = "CPU: load per core. Top: separate ps CPU average, shown below CPU; refreshed every 5s."
			case "ram":
				description = "Pressure: macOS memory pressure level or Linux 10s PSI; N/A if unavailable."
			case "network-status":
				description = "Connection type and network name are independent; name may be unavailable."
			case "network-traffic":
				description = "Upload, download and latency share one slot. Gateway ping every 5s; N/A without a response."
			}
			if config.Shown(m.draft.Visible, e.Key) && !config.EntryShown(m.draft.Visible, e.Key) {
				description = "No components enabled – hidden in the header"
			}
			rows = append(rows, row{label: e.Name, action: "toggle", entry: e, description: description})
		}
		return append(rows, row{label: "Reset menu items", action: "reset", description: "Reset items and components only; appearance stays unchanged."})
	case 1:
		rows := []row{}
		for _, t := range m.draft.ThemeOptions() {
			desc := fmt.Sprintf(m.tr("Animation: %s · Style: %s · Background: %s"), t.Animation, t.Style, config.DisplayBackground(t.Background))
			if t.Custom {
				desc += m.tr(" · Delete: remove custom theme")
			}
			mark := "[ ]"
			if m.draft.Animation == t.Animation && config.DisplayStyle(m.draft.Style) == t.Style && m.draft.Background == t.Background {
				mark = "[x]"
			}
			kind := m.tr("built-in")
			if e, ok := themepack.Lookup(t.Name); ok {
				kind = fmt.Sprintf("Download · %.0f KB", float64(e.Size)/1024)
				desc += m.tr(" · Downloaded only when you save.")
				if themepack.IsInstalled(t.Name) {
					kind = m.tr("Installed")
				}
			}
			if t.Custom {
				kind = m.tr("custom")
			}
			rows = append(rows, row{label: mark + " " + t.Name + " · " + kind, action: "theme", value: t.Name, color: t.Background, custom: t.Custom, description: desc})
		}
		rows = append(rows, row{label: "Random combination", action: "random", description: "Combine an animation with a random style and matching background."}, row{label: "Save current combination as a theme…", action: "new", description: fmt.Sprintf(m.tr("Current: %s · %s · %s. Save as a custom theme."), m.draft.Animation, config.DisplayStyle(m.draft.Style), m.draft.BackgroundName())})
		return rows
	case 2:
		return []row{
			{label: "Animation", action: "animation", value: m.draft.Animation, description: "Choose an animation; style and background stay unchanged."},
			{label: "Style", action: "style", value: config.DisplayStyle(m.draft.Style), description: "Choose header colors independently of the theme."},
			{label: "Background", action: "background", value: m.draft.BackgroundName(), color: m.draft.Background, description: "Named colors, theme backgrounds, profile color or a custom RGB color."},
		}
	default:
		return []row{{label: "Refresh interval", action: "interval", value: m.draft.RefreshInterval, description: "A positive Go duration, such as 1s, 500ms or 2.5s."}, {label: "Language", action: "language", value: languageName(m.draft.Language), description: "Choose the dashboard and settings language. Automatic follows your system locale."}}
	}
}
func (m *model) selectedRow() row {
	rows := m.rows()
	i := min(m.cursors[m.section], len(rows)-1)
	return rows[max(0, i)]
}
func (m *model) moveRow(step int) {
	rows := m.rows()
	i := m.cursors[m.section]
	for next := i + step; next >= 0 && next < len(rows); next += step {
		if !rows[next].heading {
			m.cursors[m.section] = next
			m.part = 0
			return
		}
	}
}
func (m *model) activate() tea.Cmd {
	m.notice = ""
	r := m.selectedRow()
	switch r.action {
	case "toggle":
		p := ""
		if m.part > 0 {
			p = r.entry.Parts[m.part-1]
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
				m.cursors[1] = min(m.cursors[1], len(m.rows())-1)
			case "overwrite":
				_, err := m.draft.StoreTheme(d.value)
				if err != nil {
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
	return err.Error()
}

type downloadResult struct {
	id  int
	err error
}
type catalogResult struct {
	catalog themepack.Catalog
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
		return catalogResult{catalog: c, err: err}
	}
}

func (m *model) packageLabel(name string) string {
	if e, ok := themepack.Lookup(name); ok {
		if themepack.IsInstalled(name) {
			return name + " · " + m.tr("Installed")
		}
		return fmt.Sprintf("%s · Download %.0f KB", name, float64(e.Size)/1024)
	}
	return name
}
