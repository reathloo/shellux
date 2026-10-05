package settingsui

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/reathloo/shellux/internal/config"
)

func press(m *model, key string) {
	k := tea.Key{}
	switch key {
	case "up":
		k.Code = tea.KeyUp
	case "down":
		k.Code = tea.KeyDown
	case "left":
		k.Code = tea.KeyLeft
	case "right":
		k.Code = tea.KeyRight
	case "tab":
		k.Code = tea.KeyTab
	case "shift+tab":
		k.Code = tea.KeyTab
		k.Mod = tea.ModShift
	case "enter":
		k.Code = tea.KeyEnter
	case "esc":
		k.Code = tea.KeyEscape
	case "space":
		k.Code = ' '
		k.Text = " "
	case "delete":
		k.Code = tea.KeyDelete
	case "ctrl+s":
		k.Code = 's'
		k.Mod = tea.ModCtrl
	case "ctrl+c":
		k.Code = 'c'
		k.Mod = tea.ModCtrl
	default:
		k.Code = []rune(key)[0]
		k.Text = key
	}
	m.Update(tea.KeyPressMsg(k))
}
func chooseRow(t *testing.T, m *model, action, value string) {
	t.Helper()
	m.focus = 1
	for i, r := range m.rows() {
		if r.action == action && (value == "" || r.value == value || r.entry.Key == value) {
			m.cursors[m.section] = i
			m.part = 0
			return
		}
	}
	t.Fatalf("missing row %s %s", action, value)
}

func TestKeyboardNavigationAndDraftSwitches(t *testing.T) {
	original := config.Default()
	m := newModel(original, func(config.Config) error { t.Fatal("unexpected save"); return nil })
	press(m, "tab")
	if m.focus != 1 {
		t.Fatal("tab focus")
	}
	press(m, "space")
	if m.draft.Visible["platform"] || !original.Visible["platform"] {
		t.Fatal("draft toggle")
	}
	press(m, "shift+tab")
	if m.focus != 0 {
		t.Fatal("back tab")
	}
	press(m, "down")
	if m.section != 1 {
		t.Fatal("navigation")
	}
	press(m, "up")
	chooseRow(t, m, "toggle", "cpu")
	press(m, "right")
	press(m, "space")
	if config.Shown(m.draft.Visible, "cpu.bar") || !config.Shown(m.draft.Visible, "cpu.percent") {
		t.Fatal("component focus")
	}
	press(m, "right")
	press(m, "space")
	press(m, "right")
	press(m, "space")
	if !strings.Contains(m.selectedRow().description, "No components enabled") {
		t.Fatal("missing hidden explanation")
	}
	m.part = 0
	press(m, "space")
	press(m, "space")
	if config.Shown(m.draft.Visible, "cpu.bar") {
		t.Fatal("lost component preferences")
	}
	press(m, "tab")
	if m.focus != 2 {
		t.Fatal("save button")
	}
	press(m, "right")
	if m.focus != 3 {
		t.Fatal("cancel button")
	}
	press(m, "tab")
	if m.focus != 0 {
		t.Fatal("focus wraps")
	}
	chooseRow(t, m, "reset", "")
	press(m, "enter")
	if !reflect.DeepEqual(m.draft.Visible, config.Default().Visible) {
		t.Fatal("reset")
	}
}

func TestAllNewComponentControlsRemainFocusable(t *testing.T) {
	for _, size := range [][2]int{{64, 18}, {80, 24}, {96, 30}} {
		m := newModel(config.Default(), func(config.Config) error { return nil })
		m.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		for _, name := range []string{"cpu", "ram", "battery", "network-status", "network-traffic", "date"} {
			chooseRow(t, m, "toggle", name)
			for i, part := range config.DisplayParts(name) {
				press(m, "right")
				if m.part != i+1 {
					t.Fatal("component skipped")
				}
				before := config.Shown(m.draft.Visible, name+"."+part)
				press(m, "space")
				if config.Shown(m.draft.Visible, name+"."+part) == before {
					t.Fatalf("%s.%s did not toggle", name, part)
				}
				view := m.View().Content
				if !strings.Contains(ansi.Strip(view), "Selected: "+config.PartLabel(name, part)) || !strings.Contains(view, "\x1b[") {
					t.Fatalf("focused control hidden: %s.%s at %v", name, part, size)
				}
				checkBounds(t, m, size)
				press(m, "space")
			}
			press(m, "right")
			if m.part != 0 {
				t.Fatal("component focus did not wrap")
			}
		}
	}
}

func TestCapacityErrorKeepsDraft(t *testing.T) {
	c := config.Default()
	for _, e := range config.MenuEntries() {
		if config.MenuSlots(c.Visible) == 11 {
			break
		}
		if !c.Visible[e.Key] {
			_ = c.ChangeDisplay(e.Key, "", true)
		}
	}
	m := newModel(c, func(config.Config) error { return nil })
	for _, e := range config.MenuEntries() {
		if c.Visible[e.Key] || e.Key == "time" || e.Key == "temperature" {
			continue
		}
		chooseRow(t, m, "toggle", e.Key)
		press(m, "space")
		if !strings.Contains(m.notice, "11 slots") || !reflect.DeepEqual(m.draft, c) {
			t.Fatalf("capacity validation %s: %s", e.Key, m.notice)
		}
		return
	}
	t.Fatal("no additional entry")
}

func TestDirtyCancelDefaultAndInterrupt(t *testing.T) {
	m := newModel(config.Default(), func(config.Config) error { t.Fatal("saved on cancel"); return nil })
	press(m, "esc")
	if !m.done || m.result != Cancelled {
		t.Fatal("clean cancel")
	}
	m = newModel(config.Default(), func(config.Config) error { return nil })
	press(m, "tab")
	press(m, "space")
	press(m, "esc")
	if m.modal == nil || m.modal.selected != 0 {
		t.Fatal("unsafe discard default")
	}
	press(m, "enter")
	if m.done || m.modal != nil || !m.dirty() {
		t.Fatal("continue editing")
	}
	press(m, "esc")
	press(m, "right")
	press(m, "enter")
	if !m.done || m.result != Cancelled {
		t.Fatal("discard")
	}
	m = newModel(config.Default(), func(config.Config) error { return nil })
	_ = m.openInput("Interval", "interval", "1s", "")
	press(m, "ctrl+c")
	if !m.done || m.result != Interrupted {
		t.Fatal("interrupt")
	}
}

func TestSelectionInputAndSaveFailure(t *testing.T) {
	saves := 0
	m := newModel(config.Default(), func(config.Config) error { saves++; return config.ErrChanged })
	m.section = 2
	chooseRow(t, m, "style", "")
	press(m, "enter")
	if m.modal == nil || m.modal.kind != "select" {
		t.Fatal("style selector")
	}
	for i, choice := range m.modal.choices {
		if choice.value == "neon" {
			m.modal.selected = i
		}
	}
	press(m, "enter")
	if m.draft.Style != "neon" {
		t.Fatal("style selection")
	}
	chooseRow(t, m, "background", "")
	press(m, "enter")
	m.modal.selected = len(m.modal.choices) - 1
	press(m, "enter")
	m.modal.input.SetValue("#xyz")
	press(m, "enter")
	if m.modal == nil || m.modal.error == "" || m.draft.Background != m.original.Background {
		t.Fatal("invalid RGB changed draft")
	}
	m.modal.input.SetValue("#123456")
	press(m, "enter")
	if m.draft.Background != "#123456" {
		t.Fatal("RGB input")
	}
	m.section = 3
	chooseRow(t, m, "interval", "")
	press(m, "enter")
	m.modal.input.SetValue("0s")
	press(m, "enter")
	if m.modal == nil || m.modal.error == "" {
		t.Fatal("invalid duration")
	}
	m.modal.input.SetValue("")
	m.Update(tea.PasteMsg{Content: "500ms"})
	press(m, "enter")
	if m.draft.RefreshInterval != "500ms" {
		t.Fatal("paste")
	}
	press(m, "ctrl+s")
	if saves != 1 || m.done || !strings.Contains(m.notice, "changed externally") {
		t.Fatal("conflict closed UI")
	}
	if !strings.Contains(ansi.Strip(m.View().Content), "reopen") || !strings.Contains(ansi.Strip(m.View().Content), "settings.") {
		t.Fatalf("conflict instruction invisible: %q", ansi.Strip(m.View().Content))
	}
	m.save = func(c config.Config) error {
		if c.RefreshInterval != "500ms" {
			t.Fatal("wrong saved draft")
		}
		return nil
	}
	press(m, "ctrl+s")
	if !m.done || m.result != Saved {
		t.Fatal("save")
	}
}

func TestOwnThemeDialogsProtectAndDeferChanges(t *testing.T) {
	m := newModel(config.Default(), func(config.Config) error { return nil })
	m.section = 1
	chooseRow(t, m, "new", "")
	press(m, "enter")
	m.modal.input.SetValue("purplemonster")
	press(m, "enter")
	if m.modal == nil || m.modal.error == "" || len(m.draft.Themes) != 0 {
		t.Fatal("overwrote built-in")
	}
	m.modal.input.SetValue("my-theme")
	press(m, "enter")
	if len(m.draft.Themes) != 1 {
		t.Fatal("theme not created")
	}
	old := m.draft.Themes["my-theme"]
	_ = m.draft.ChangeAppearance("style", "neon")
	chooseRow(t, m, "new", "")
	press(m, "enter")
	m.modal.input.SetValue("my-theme")
	press(m, "enter")
	if m.modal.kind != "confirm" || m.modal.selected != 0 || m.draft.Themes["my-theme"] != old {
		t.Fatal("overwrite without confirmation")
	}
	press(m, "esc")
	chooseRow(t, m, "new", "")
	press(m, "enter")
	m.modal.input.SetValue("my-theme")
	press(m, "enter")
	press(m, "right")
	press(m, "enter")
	if m.draft.Themes["my-theme"].Style != "neon" {
		t.Fatal("confirmed overwrite")
	}
	chooseRow(t, m, "theme", "my-theme")
	press(m, "delete")
	if m.modal == nil || len(m.draft.Themes) != 1 {
		t.Fatal("delete without confirmation")
	}
	press(m, "right")
	press(m, "enter")
	if len(m.draft.Themes) != 0 {
		t.Fatal("confirmed delete")
	}
	chooseRow(t, m, "theme", "purplemonster")
	press(m, "delete")
	if m.modal != nil || m.notice == "" {
		t.Fatal("built-in deletion allowed")
	}
}

func TestViewBoundsScrollAndFocus(t *testing.T) {
	m := newModel(config.Default(), func(config.Config) error { return nil })
	for i := range 40 {
		_, _ = m.draft.StoreTheme(fmt.Sprintf("long-theme-name-with-suffix-%02d", i))
	}
	for _, size := range [][2]int{{140, 40}, {96, 30}, {64, 18}, {63, 17}, {30, 8}, {1, 1}} {
		m.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		for section := range 4 {
			m.section = section
			m.focus = 1
			m.cursors[section] = len(m.rows()) - 1
			checkBounds(t, m, size)
			if !m.small() && !strings.Contains(ansi.Strip(m.View().Content), "› "+ansi.Truncate(m.selectedRow().label, 20, "")) {
				t.Fatalf("selected row invisible at %v section %d", size, section)
			}
		}
		m.section = 2
		chooseRow(t, m, "background", "")
		press(m, "enter")
		if m.modal != nil {
			m.modal.selected = len(m.modal.choices) - 1
			checkBounds(t, m, size)
			m.modal = nil
		}
	}
}
func checkBounds(t *testing.T, m *model, size [2]int) {
	t.Helper()
	v := m.View()
	if !v.AltScreen || v.WindowTitle != "Shellux" {
		t.Fatal("view screen settings")
	}
	lines := strings.Split(v.Content, "\n")
	if len(lines) > size[1] {
		t.Fatalf("height %d > %d", len(lines), size[1])
	}
	for _, l := range lines {
		if w := lipgloss.Width(l); w > size[0] {
			t.Fatalf("width %d > %d: %q", w, size[0], l)
		}
	}
}

func TestSmallWindowRetainsDraftAndAllowsDiscard(t *testing.T) {
	m := newModel(config.Default(), func(config.Config) error { return nil })
	press(m, "tab")
	press(m, "space")
	before := m.draft.Clone()
	m.Update(tea.WindowSizeMsg{Width: 40, Height: 10})
	press(m, "down")
	press(m, "space")
	press(m, "ctrl+s")
	if !reflect.DeepEqual(m.draft, before) || m.done {
		t.Fatal("small window changed or saved draft")
	}
	press(m, "esc")
	if m.modal == nil || m.modal.selected != 0 {
		t.Fatal("small discard prompt")
	}
	press(m, "right")
	press(m, "enter")
	if !m.done || m.result != Cancelled {
		t.Fatal("small cancel")
	}
}

func TestEditingCancelDoesNotWriteSession(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	session, err := config.OpenEditor()
	if err != nil {
		t.Fatal(err)
	}
	m := newModel(session.Config, session.Save)
	press(m, "tab")
	press(m, "space")
	press(m, "esc")
	press(m, "right")
	press(m, "enter")
	path, _ := config.DefaultPath()
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("cancel wrote config")
	}
}

func TestFocusedSaveButtonRetriesWriteFailure(t *testing.T) {
	attempts := 0
	m := newModel(config.Default(), func(config.Config) error {
		attempts++
		if attempts == 1 {
			return os.ErrPermission
		}
		return nil
	})
	press(m, "tab")
	press(m, "space")
	press(m, "tab")
	press(m, "enter")
	if m.done || m.focus != 2 || !m.dirty() || !strings.Contains(m.notice, "Could not save") {
		t.Fatal("write failure lost draft or button focus")
	}
	press(m, "enter")
	if !m.done || m.result != Saved || attempts != 2 {
		t.Fatal("save button retry")
	}
}
