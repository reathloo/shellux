package settingsui

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/reathloo/shellux/internal/config"
	"github.com/reathloo/shellux/internal/themepack"
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
	case "shift+up":
		k.Code = tea.KeyUp
		k.Mod = tea.ModShift
	case "shift+down":
		k.Code = tea.KeyDown
		k.Mod = tea.ModShift
	case "shift+left":
		k.Code = tea.KeyLeft
		k.Mod = tea.ModShift
	case "shift+right":
		k.Code = tea.KeyRight
		k.Mod = tea.ModShift
	case "ctrl+up":
		k.Code = tea.KeyUp
		k.Mod = tea.ModCtrl
	case "ctrl+down":
		k.Code = tea.KeyDown
		k.Mod = tea.ModCtrl
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

func TestThemeFavoriteAndUninstallControls(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	entry, ok := themepack.Lookup("purplemonster")
	if !ok {
		t.Fatal("missing catalog theme")
	}
	archive, err := os.ReadFile("../../themes/" + entry.Package)
	if err != nil {
		t.Fatal(err)
	}
	store, err := themepack.DefaultStore()
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Import(entry, archive); err != nil {
		t.Fatal(err)
	}
	m := newModel(config.Default(), func(config.Config) error { return nil })
	m.section = themesSection
	chooseRow(t, m, "theme", entry.Name)
	press(m, "f")
	if !m.draft.FavoriteThemes[entry.Name] || m.original.FavoriteThemes[entry.Name] || m.selectedRow().value != entry.Name {
		t.Fatal("favorite did not stay selected in the draft")
	}
	press(m, "u")
	if m.modal == nil || m.modal.action != "uninstall" {
		t.Fatal("uninstall confirmation missing")
	}
	press(m, "right")
	press(m, "enter")
	if themepack.IsInstalled(entry.Name) || m.modal != nil {
		t.Fatal("theme was not uninstalled")
	}
}

func TestThemeInstallStatusIsCheckedOncePerSettingsSession(t *testing.T) {
	m := newModel(config.Default(), nil)
	m.section = themesSection
	calls := map[string]int{}
	m.checkInstalled = func(name string) bool {
		calls[name]++
		return name == "purplemonster"
	}
	for i := 0; i < 3; i++ {
		m.rows()
		m.View()
		m.packageLabel("purplemonster")
	}
	if calls["purplemonster"] != 1 {
		t.Fatalf("theme package checked %d times", calls["purplemonster"])
	}
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
	if m.section != presetsSection {
		t.Fatal("navigation")
	}
	press(m, "up")
	chooseRow(t, m, "toggle", "cpu")
	m.part = slices.Index(m.selectedRow().entry.Parts, "bar") + 2
	press(m, "space")
	if config.Shown(m.draft.Visible, "cpu.bar") || !config.Shown(m.draft.Visible, "cpu.percent") {
		t.Fatal("component focus")
	}
	m.part = slices.Index(m.selectedRow().entry.Parts, "percent") + 2
	press(m, "space")
	m.part = slices.Index(m.selectedRow().entry.Parts, "value") + 2
	press(m, "space")
	if config.EntryShown(m.draft.Visible, "cpu") || !strings.Contains(m.menuHelp("cpu", "value"), "one-minute CPU load") {
		t.Fatal("component help or hidden state is incorrect")
	}
	m.part = 1
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

func TestKeyboardReordersGroupsItemsAndComponents(t *testing.T) {
	m := newModel(config.Default(), func(config.Config) error { return nil })
	m.focus = 1
	chooseRow(t, m, "group", "RESOURCES")
	press(m, "shift+up")
	press(m, "shift+up")
	if m.draft.MenuLayout.Groups[0] != "RESOURCES" || m.selectedRow().value != "RESOURCES" {
		t.Fatalf("group move failed: %v", m.draft.MenuLayout.Groups)
	}
	chooseRow(t, m, "toggle", "cpu")
	press(m, "shift+down")
	if m.draft.MenuLayout.Items["RESOURCES"][1] != "cpu" || m.selectedRow().entry.Key != "cpu" {
		t.Fatalf("item move failed: %v", m.draft.MenuLayout.Items["RESOURCES"])
	}
	m.part = slices.Index(m.selectedRow().entry.Parts, "sparkline") + 2
	press(m, "shift+right")
	press(m, "shift+right")
	press(m, "shift+left")
	if m.draft.MenuLayout.Components["cpu"][1] != "sparkline" || m.part != 3 {
		t.Fatalf("component move failed: %v, focus %d", m.draft.MenuLayout.Components["cpu"], m.part)
	}
	if !strings.Contains(m.notice, "Save") {
		t.Fatal("reorder lacks save notice")
	}
	chooseRow(t, m, "reset", "")
	press(m, "enter")
	if !reflect.DeepEqual(m.draft.MenuLayout, config.Default().MenuLayout) {
		t.Fatal("settings reset did not restore layout")
	}
}

func TestItemMoveWorksFromSwitchFocusAndWithoutShift(t *testing.T) {
	m := newModel(config.Default(), func(config.Config) error { return nil })
	chooseRow(t, m, "toggle", "cpu")
	m.part = 1 // Enabled, not the item name
	press(m, "shift+down")
	if got := m.draft.MenuLayout.Items["RESOURCES"][:2]; !slices.Equal(got, []string{"cores", "cpu"}) {
		t.Fatalf("Shift+Down on Enabled did not move CPU: %v", got)
	}
	m.part = 2 // first detail switch
	press(m, "shift+up")
	if got := m.draft.MenuLayout.Items["RESOURCES"][:2]; !slices.Equal(got, []string{"cpu", "cores"}) {
		t.Fatalf("Shift+Up on a detail did not move CPU: %v", got)
	}
	press(m, "m")
	if !m.moveMode {
		t.Fatal("move mode did not open")
	}
	press(m, "down")
	if got := m.draft.MenuLayout.Items["RESOURCES"][:2]; !slices.Equal(got, []string{"cores", "cpu"}) {
		t.Fatalf("ordinary Down in move mode did not move CPU: %v", got)
	}
	press(m, "right")
	if got := m.draft.MenuLayout.Components["cpu"][:2]; !slices.Equal(got, []string{"bar", "sparkline"}) {
		t.Fatalf("ordinary Right in move mode did not move sparkline: %v", got)
	}
	press(m, "enter")
	if m.moveMode || m.done {
		t.Fatal("Enter did not leave move mode without closing settings")
	}
}

func TestEveryComponentCanMoveInBothDirections(t *testing.T) {
	for _, entry := range config.MenuEntries() {
		if len(entry.Parts) < 2 {
			continue
		}
		for index, part := range entry.Parts {
			m := newModel(config.Default(), func(config.Config) error { return nil })
			chooseRow(t, m, "toggle", entry.Key)
			m.part = index + 2
			if index < len(entry.Parts)-1 {
				press(m, "shift+right")
				press(m, "shift+left")
			} else {
				press(m, "shift+left")
				press(m, "shift+right")
			}
			if !slices.Equal(m.draft.MenuLayout.Components[entry.Key], entry.Parts) || m.part != index+2 {
				t.Fatalf("%s.%s did not move both ways: %v, focus %d", entry.Key, part, m.draft.MenuLayout.Components[entry.Key], m.part)
			}
		}
	}
}

func TestAllNewComponentControlsRemainFocusable(t *testing.T) {
	for _, size := range [][2]int{{64, 18}, {80, 24}, {96, 30}} {
		m := newModel(config.Default(), func(config.Config) error { return nil })
		m.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		for _, name := range []string{"cpu", "ram", "battery", "network-status", "network-traffic", "date", "disk-io", "packet-loss", "logged-users", "media-activity"} {
			chooseRow(t, m, "toggle", name)
			ordered := append([]string(nil), m.selectedRow().entry.Parts...)
			for i, part := range ordered {
				m.part = i + 2
				before := config.Shown(m.draft.Visible, name+"."+part)
				press(m, "space")
				if config.Shown(m.draft.Visible, name+"."+part) == before {
					t.Fatalf("%s.%s did not toggle", name, part)
				}
				view := m.View().Content
				if !strings.Contains(ansi.Strip(view), m.componentPreview(name, part)) || m.menuHelp(name, part) == "" || !strings.Contains(view, "\x1b[") {
					t.Fatalf("focused control hidden: %s.%s at %v", name, part, size)
				}
				checkBounds(t, m, size)
				press(m, "space")
			}
			m.part = len(ordered) + 1
			press(m, "right")
			if m.part != 0 {
				t.Fatal("component focus did not wrap")
			}
		}
	}
}

func TestEveryMenuControlHasSpecificLocalizedHelp(t *testing.T) {
	m := newModel(config.Default(), func(config.Config) error { return nil })
	english := map[string]string{}
	englishByControl := map[string]string{}
	for _, entry := range config.MenuEntries() {
		keys := append([]string{""}, entry.Parts...)
		for _, part := range keys {
			text := m.menuHelp(entry.Key, part)
			if text == "" {
				t.Fatalf("missing English help for %s.%s", entry.Key, part)
			}
			if previous := english[text]; previous != "" {
				t.Fatalf("duplicate English help for %s.%s and %s", entry.Key, part, previous)
			}
			control := entry.Key + "." + part
			english[text] = control
			englishByControl[control] = text
		}
	}

	m.draft.Language = "de"
	for _, entry := range config.MenuEntries() {
		for _, part := range append([]string{""}, entry.Parts...) {
			control := entry.Key + "." + part
			if text := m.menuHelp(entry.Key, part); text == "" || text == englishByControl[control] {
				t.Fatalf("missing German help for %s.%s: %q", entry.Key, part, text)
			}
		}
	}
}

func TestComponentControlsShowOutputPreviews(t *testing.T) {
	m := newModel(config.Default(), func(config.Config) error { return nil })
	chooseRow(t, m, "toggle", "cpu")
	if view := ansi.Strip(m.View().Content); !strings.Contains(view, "[✓]") || !strings.Contains(view, "[ ]") {
		t.Fatalf("settings view does not distinguish enabled and disabled controls: %s", view)
	}
	previews := map[string]string{
		"bar": "████░░░░", "sparkline": "▂▃▅▇▆▄▂▃", "percent": "42%",
		"top": "code (24%)", "pressure": "Normal", "remaining": "~3h 20m",
		"upload": "↑ 1.4 MB/s", "download": "↓ 8.2 MB/s", "latency": "18 ms",
	}
	for part, want := range previews {
		entry := "cpu"
		if part == "pressure" {
			entry = "ram"
		}
		if part == "remaining" {
			entry = "battery"
		}
		if part == "upload" || part == "download" || part == "latency" {
			entry = "network-traffic"
		}
		if got := m.componentPreview(entry, part); got != want {
			t.Fatalf("%s preview = %q, want %q", part, got, want)
		}
	}

	m.draft.Language = "de"
	if got := m.componentPreview("network-status", "value"); got != "Verbunden" {
		t.Fatalf("German status preview = %q", got)
	}
	if got := m.componentPreview("date", "value"); got != "15.10.2026" {
		t.Fatalf("German date preview = %q", got)
	}
}

func TestEveryEntryShowsAnOutputPreview(t *testing.T) {
	m := newModel(config.Default(), func(config.Config) error { return nil })
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	seen := map[string]string{}
	for _, entry := range config.MenuEntries() {
		preview := m.entryPreview(entry.Key)
		if preview == "" || preview == config.DisplayMenuItem(entry.Key) || preview == "Enabled" {
			t.Fatalf("%s has no concrete preview: %q", entry.Key, preview)
		}
		if previous := seen[preview]; previous != "" {
			t.Fatalf("%s and %s share preview %q", entry.Key, previous, preview)
		}
		seen[preview] = entry.Key
		chooseRow(t, m, "toggle", entry.Key)
		view := ansi.Strip(m.View().Content)
		if len(entry.Parts) == 0 && !strings.Contains(view, preview) {
			t.Fatalf("%s has no detail switches, but its preview is missing: %s", entry.Key, view)
		}
		if len(entry.Parts) > 0 && strings.Contains(view, "Enabled · "+preview) {
			t.Fatalf("%s repeats its preview before the detail switches: %s", entry.Key, view)
		}
		for _, part := range entry.Parts {
			partPreview := m.componentPreview(entry.Key, part)
			if partPreview == "" || partPreview == config.PartLabel(entry.Key, part) {
				t.Fatalf("%s.%s has no concrete preview: %q", entry.Key, part, partPreview)
			}
		}
	}

	chooseRow(t, m, "toggle", "uptime")
	view := ansi.Strip(m.View().Content)
	if !strings.Contains(view, "[✓] Enabled · 2h 5m") {
		t.Fatalf("entry preview is not rendered as its switch label: %s", view)
	}
	m.draft.Language = "de"
	if view := ansi.Strip(m.View().Content); !strings.Contains(view, "[✓] Aktiviert · 2h 5m") {
		t.Fatalf("German master switch is missing: %s", view)
	}
	chooseRow(t, m, "toggle", "cpu")
	if view := ansi.Strip(m.View().Content); !strings.Contains(view, "[✓] Aktiviert") || strings.Contains(view, "Aktiviert · ████░░░░ 42%") {
		t.Fatalf("resource master switch still repeats its preview: %s", view)
	}
}

func TestHelpTextFollowsFocusedControl(t *testing.T) {
	m := newModel(config.Default(), func(config.Config) error { return nil })
	m.width, m.height = 160, 30
	chooseRow(t, m, "toggle", "cpu")

	entryHelp := m.menuHelp("cpu", "")
	if view := ansi.Strip(m.View().Content); !strings.Contains(view, entryHelp) {
		t.Fatalf("CPU entry help is not visible: %s", view)
	}
	press(m, "right")
	m.part = slices.Index(m.selectedRow().entry.Parts, "bar") + 2
	barHelp := m.menuHelp("cpu", "bar")
	if view := ansi.Strip(m.View().Content); !strings.Contains(view, barHelp) || strings.Contains(view, entryHelp) {
		t.Fatalf("help did not follow bar focus: %s", view)
	}
	m.part = slices.Index(m.selectedRow().entry.Parts, "percent") + 2
	percentHelp := m.menuHelp("cpu", "percent")
	if view := ansi.Strip(m.View().Content); !strings.Contains(view, percentHelp) || strings.Contains(view, barHelp) {
		t.Fatalf("help did not follow percent focus: %s", view)
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
	m.section = appearanceSection
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
	m.section = generalSection
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
	m.section = themesSection
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

func TestMenuPresetDialogsAndDraftSelection(t *testing.T) {
	var saved config.Config
	saves := 0
	m := newModel(config.Default(), func(c config.Config) error { saved = c; saves++; return nil })
	m.section = presetsSection
	chooseRow(t, m, "new-preset", "")
	press(m, "enter")
	m.modal.input.SetValue("work")
	press(m, "enter")
	if len(m.draft.MenuPresets) != 1 || saves != 0 || m.original.MenuPresets != nil {
		t.Fatal("preset creation wrote outside the draft")
	}
	m.section = menuSection
	chooseRow(t, m, "toggle", "cpu")
	press(m, "shift+down")
	m.section = presetsSection
	chooseRow(t, m, "new-preset", "")
	press(m, "enter")
	m.modal.input.SetValue("work")
	press(m, "enter")
	if m.modal == nil || m.modal.kind != "confirm" || m.draft.MenuPresets["work"].Layout.Items["RESOURCES"][0] != "cpu" {
		t.Fatal("preset overwritten without confirmation")
	}
	press(m, "right")
	press(m, "enter")
	if m.draft.MenuPresets["work"].Layout.Items["RESOURCES"][0] != "cores" {
		t.Fatal("confirmed preset overwrite did not store current order")
	}
	m.draft.Style = "neon"
	chooseRow(t, m, "preset", "default")
	press(m, "enter")
	if !m.draft.MenuPresetMatches("default") || m.draft.Style != "neon" {
		t.Fatal("default preset changed theme")
	}
	chooseRow(t, m, "preset", "work")
	press(m, "enter")
	if !m.draft.MenuPresetMatches("work") || m.draft.Style != "neon" {
		t.Fatal("saved preset did not apply independently")
	}
	press(m, "ctrl+s")
	if saves != 1 || saved.MenuPresets["work"].Layout.Items["RESOURCES"][0] != "cores" {
		t.Fatal("preset was not saved with the menu")
	}

	m = newModel(saved, func(c config.Config) error { saved = c; return nil })
	m.section = presetsSection
	chooseRow(t, m, "preset", "work")
	press(m, "delete")
	if m.modal == nil || len(m.draft.MenuPresets) != 1 {
		t.Fatal("preset deletion skipped confirmation")
	}
	press(m, "right")
	press(m, "enter")
	if len(m.draft.MenuPresets) != 0 || m.draft.MenuLayout.Items["RESOURCES"][0] != "cores" || m.draft.Style != "neon" {
		t.Fatal("preset deletion changed active settings")
	}
}

func TestViewBoundsScrollAndFocus(t *testing.T) {
	m := newModel(config.Default(), func(config.Config) error { return nil })
	for i := range 40 {
		_, _ = m.draft.StoreTheme(fmt.Sprintf("long-theme-name-with-suffix-%02d", i))
	}
	for _, size := range [][2]int{{140, 40}, {96, 30}, {64, 18}, {63, 17}, {30, 8}, {1, 1}} {
		m.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		for section := range sectionCount {
			m.section = section
			m.focus = 1
			m.cursors[section] = len(m.rows()) - 1
			checkBounds(t, m, size)
			if !m.small() && !strings.Contains(ansi.Strip(m.View().Content), "› "+ansi.Truncate(m.selectedRow().label, 20, "")) {
				t.Fatalf("selected row invisible at %v section %d", size, section)
			}
		}
		m.section = appearanceSection
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
