package config

import (
	"fmt"
	"maps"
	"slices"
)

// menuItems is the single source of menu names, public labels and defaults.
var menuItems = []struct {
	name, display, group string
	visible, parts       bool
}{
	{"platform", "platform", "SYSTEM", true, false},
	{"shell", "shell", "SYSTEM", true, false},
	{"uptime", "uptime", "SYSTEM", true, false},
	{"date", "date", "SYSTEM", false, true},
	{"hostname", "hostname", "SYSTEM", false, false},
	{"media-activity", "media", "SYSTEM", false, true},
	{"network-status", "status", "NETWORK", true, true},
	{"network-traffic", "traffic", "NETWORK", true, true},
	{"ip", "ip", "NETWORK", false, false},
	{"packet-loss", "loss", "NETWORK", false, true},
	{"cpu", "cpu", "RESOURCES", true, true},
	{"cores", "cores", "RESOURCES", false, true},
	{"ram", "ram", "RESOURCES", true, true},
	{"load-average", "load", "RESOURCES", false, true},
	{"swap", "swap", "RESOURCES", false, true},
	{"gpu", "gpu", "RESOURCES", false, true},
	{"volume", "volume", "RESOURCES", true, true},
	{"disk-io", "disk I/O", "RESOURCES", false, true},
	{"battery", "battery", "RESOURCES", true, true},
	{"temperature", "thermal", "RESOURCES", false, false},
	{"directory", "path", "SESSION", true, false},
	{"spotify", "spotify", "SESSION", true, false},
	{"progress", "progress", "SESSION", true, false},
	{"logged-users", "users", "SESSION", false, true},
	{"time", "time", "HEADER", true, false},
}

// MenuLayout stores the independent order of dashboard groups, their entries,
// and the components inside each entry. Visibility stays separate so moving a
// disabled control does not silently enable it.
type MenuLayout struct {
	Groups     []string            `json:"groups"`
	Items      map[string][]string `json:"items"`
	Components map[string][]string `json:"components"`
}

func defaultMenuLayout() MenuLayout {
	layout := MenuLayout{Items: map[string][]string{}, Components: map[string][]string{}}
	for _, item := range menuItems {
		if !slices.Contains(layout.Groups, item.group) {
			layout.Groups = append(layout.Groups, item.group)
		}
		layout.Items[item.group] = append(layout.Items[item.group], item.name)
		if parts := defaultComponentOrder(item.name); len(parts) > 0 {
			layout.Components[item.name] = parts
		}
	}
	return layout
}

func defaultComponentOrder(name string) []string {
	switch canonicalMenuItem(name) {
	case "cpu":
		return []string{"sparkline", "bar", "value", "percent", "top"}
	case "ram":
		return []string{"sparkline", "bar", "value", "percent", "pressure"}
	case "battery":
		return []string{"sparkline", "bar", "percent", "value", "remaining"}
	case "network-traffic":
		return []string{"sparkline", "upload", "download", "latency"}
	case "network-status":
		return []string{"value", "connection", "name", "interface", "ip"}
	case "cores":
		return []string{"bar", "value", "percent"}
	case "volume":
		return []string{"bar", "value", "percent", "remaining"}
	default:
		return DisplayParts(name)
	}
}

func cloneStringSlices(source map[string][]string) map[string][]string {
	if source == nil {
		return nil
	}
	result := maps.Clone(source)
	for key, values := range result {
		result[key] = append([]string(nil), values...)
	}
	return result
}

// completeMenuLayout migrates configurations written before ordering existed
// and fills fields added by newer Shellux versions without changing old choices.
func completeMenuLayout(layout MenuLayout) MenuLayout {
	layout.Groups = append([]string(nil), layout.Groups...)
	layout.Items = cloneStringSlices(layout.Items)
	layout.Components = cloneStringSlices(layout.Components)
	defaults := defaultMenuLayout()
	if len(layout.Groups) == 0 {
		layout.Groups = defaults.Groups
	}
	if layout.Items == nil {
		layout.Items = map[string][]string{}
	}
	for group, items := range defaults.Items {
		if len(layout.Items[group]) == 0 {
			layout.Items[group] = items
			continue
		}
		for _, name := range items {
			if !slices.Contains(layout.Items[group], name) {
				layout.Items[group] = append(layout.Items[group], name)
			}
		}
	}
	if layout.Components == nil {
		layout.Components = map[string][]string{}
	}
	for entry, parts := range defaults.Components {
		if len(layout.Components[entry]) == 0 {
			layout.Components[entry] = parts
			continue
		}
		for _, part := range parts {
			if !slices.Contains(layout.Components[entry], part) {
				layout.Components[entry] = append(layout.Components[entry], part)
			}
		}
	}
	return layout
}

func validatePermutation(label string, got, want []string) error {
	if len(got) != len(want) {
		return fmt.Errorf("%s must contain every supported value exactly once", label)
	}
	seen := map[string]bool{}
	for _, value := range got {
		if seen[value] || !slices.Contains(want, value) {
			return fmt.Errorf("%s contains invalid or duplicate value %q", label, value)
		}
		seen[value] = true
	}
	return nil
}

func validateMenuLayout(layout MenuLayout) error {
	defaults := defaultMenuLayout()
	if err := validatePermutation("menu groups", layout.Groups, defaults.Groups); err != nil {
		return err
	}
	if len(layout.Items) != len(defaults.Items) || len(layout.Components) != len(defaults.Components) {
		return fmt.Errorf("menu layout contains unsupported entries")
	}
	for group, want := range defaults.Items {
		if err := validatePermutation("menu group "+group, layout.Items[group], want); err != nil {
			return err
		}
	}
	for entry, want := range defaults.Components {
		if err := validatePermutation("components for "+entry, layout.Components[entry], want); err != nil {
			return err
		}
	}
	return nil
}

// MenuItems returns supported config keys in display order.
func MenuItems() []string {
	names := make([]string, 0, len(menuItems))
	for _, item := range menuItems {
		names = append(names, item.name)
	}
	return names
}

func defaultVisibility() map[string]bool {
	visible := make(map[string]bool, len(menuItems))
	for _, item := range menuItems {
		visible[item.name] = item.visible
	}
	return visible
}

// UpdateMenu changes the selection without changing appearance or refresh settings.
func UpdateMenu(action string, names []string) (string, error) {
	settings, err := LoadDefault()
	if err != nil {
		return "", err
	}
	names = append([]string(nil), names...)
	for index, name := range names {
		names[index] = canonicalMenuItem(name)
	}
	for _, name := range names {
		if !contains(MenuItems(), name) {
			return "", fmt.Errorf("unknown menu item %q", name)
		}
	}
	switch action {
	case "set":
		for _, name := range MenuItems() {
			settings.Visible[name] = false
		}
		for _, name := range names {
			if settings.Visible[name] {
				return "", fmt.Errorf("duplicate menu item %q", name)
			}
			settings.Visible[name] = true
		}
	case "replace":
		if len(names) != 2 {
			return "", fmt.Errorf("show replace requires two items")
		}
		if !settings.Visible[names[0]] {
			return "", fmt.Errorf("menu item %q is not selected", names[0])
		}
		if names[0] != names[1] && settings.Visible[names[1]] {
			return "", fmt.Errorf("menu item %q is already selected", names[1])
		}
		settings.Visible[names[0]] = false
		settings.Visible[names[1]] = true
	case "reset":
		if len(names) != 0 {
			return "", fmt.Errorf("menu reset takes no items")
		}
		settings.ResetMenu()
	default:
		return "", fmt.Errorf("unknown menu action %q", action)
	}
	if err := validateMenu(settings.Visible); err != nil {
		return "", err
	}
	return writeDefault(settings)
}

// MenuSlots counts effective header entries, including shared slots.
func MenuSlots(visible map[string]bool) int {
	count := 0
	for _, name := range MenuItems() {
		if !EntryShown(visible, name) || name == "time" {
			continue
		}
		// Battery and thermal pressure share a single dashboard entry.
		if name == "temperature" && EntryShown(visible, "battery") {
			continue
		}
		// Spotify title and progress occupy the two rows formerly owned by playback.
		if name == "progress" && EntryShown(visible, "spotify") {
			continue
		}
		count++
	}
	return count
}

func validateMenu(visible map[string]bool) error {
	if MenuSlots(visible) > 11 {
		return fmt.Errorf("menu supports at most 11 slots; hide or replace an item first")
	}
	return nil
}

// DisplayParts returns the independently configurable components of an entry.
func DisplayParts(name string) []string {
	name = canonicalMenuItem(name)
	switch name {
	case "date":
		return []string{"value", "timezone", "utc"}
	case "cpu":
		return []string{"bar", "percent", "value", "top", "sparkline"}
	case "ram":
		return []string{"bar", "percent", "value", "pressure", "sparkline"}
	case "battery":
		return []string{"bar", "percent", "value", "remaining", "sparkline"}
	case "network-traffic":
		return []string{"upload", "download", "latency", "sparkline"}
	case "network-status":
		return []string{"value", "interface", "ip", "connection", "name"}
	case "disk-io":
		return []string{"read", "write"}
	case "load-average":
		return []string{"one", "five", "fifteen"}
	case "swap":
		return []string{"value", "percent", "remaining"}
	case "gpu":
		return []string{"name", "cores"}
	case "volume":
		return []string{"bar", "percent", "value", "remaining"}
	case "packet-loss":
		return []string{"percent", "samples"}
	case "logged-users":
		return []string{"count", "names"}
	case "media-activity":
		return []string{"microphone", "camera"}
	}
	for _, item := range menuItems {
		if item.name == name && item.parts {
			return []string{"bar", "percent", "value"}
		}
	}
	return nil
}

// EntryShown applies the same component rules to rendering and slot counting.
func EntryShown(visible map[string]bool, name string) bool {
	name = canonicalMenuItem(name)
	if !Shown(visible, name) {
		return false
	}
	parts := DisplayParts(name)
	if len(parts) == 0 {
		return true
	}
	for _, part := range parts {
		if Shown(visible, name+"."+part) {
			return true
		}
	}
	return false
}

// PartLabel supplies readable labels for CLI-independent UI controls.
func PartLabel(name, part string) string {
	if part == "remaining" && canonicalMenuItem(name) == "volume" {
		return "Free space"
	}
	if part == "remaining" && canonicalMenuItem(name) == "swap" {
		return "Free swap"
	}
	if part == "name" && canonicalMenuItem(name) == "gpu" {
		return "GPU model"
	}
	if canonicalMenuItem(name) == "date" && part == "value" {
		return "Date"
	}
	if canonicalMenuItem(name) == "battery" && part == "value" {
		return "Charging"
	}
	if canonicalMenuItem(name) == "network-status" && part == "value" {
		return "Status"
	}
	labels := map[string]string{
		"bar": "Bar", "sparkline": "History", "percent": "Percent", "value": "Value", "top": "Top process",
		"pressure": "Pressure", "remaining": "Remaining", "interface": "Interface",
		"ip": "IP", "connection": "Connection", "name": "Network name",
		"timezone": "Timezone", "utc": "UTC offset",
		"upload": "Upload", "download": "Download", "latency": "Latency",
		"read": "Read", "write": "Write", "samples": "Samples",
		"count": "Count", "names": "Users", "microphone": "Microphone", "camera": "Camera",
		"one": "1 min", "five": "5 min", "fifteen": "15 min", "cores": "Cores",
	}
	return labels[part]
}

// Shown preserves original defaults while leaving newly added details off.
func Shown(visible map[string]bool, name string) bool {
	value, configured := visible[name]
	if configured {
		return value
	}
	for _, item := range menuItems {
		if item.name == name {
			// Older rendering callers enable original metrics through a nil map.
			return item.visible || name == "temperature"
		}
	}
	switch name {
	case "cpu.sparkline", "cpu.top", "ram.sparkline", "ram.pressure", "battery.bar", "battery.sparkline", "battery.remaining", "volume.remaining", "network-status.connection", "network-status.name", "date.timezone", "date.utc", "network-traffic.sparkline", "network-traffic.latency":
		return false // New optional details preserve the existing header.
	}
	return true // Original component switches default to on.
}

// SetDisplay toggles a whole entry or one component, preserving other choices.
func SetDisplay(name, part string, visible bool) (string, error) {
	settings, err := LoadDefault()
	if err != nil {
		return "", err
	}
	if err := settings.ChangeDisplay(name, part, visible); err != nil {
		return "", err
	}
	return writeDefault(settings)
}

// ChangeDisplay validates a switch before replacing the draft, preserving details.
func (c *Config) ChangeDisplay(name, part string, visible bool) error {
	name = canonicalMenuItem(name)
	if !contains(MenuItems(), name) {
		return fmt.Errorf("unknown menu item %q", name)
	}
	key := name
	if part != "" {
		if !contains(DisplayParts(name), part) {
			return fmt.Errorf("item %q does not support part %q; supported parts: %v", name, part, DisplayParts(name))
		}
		key += "." + part
	}
	next := c.Clone()
	next.Visible[key] = visible
	if err := next.Validate(); err != nil {
		return err
	}
	*c = next
	return nil
}

// DisplayMenuItem matches command names to their dashboard labels.
func DisplayMenuItem(name string) string {
	for _, item := range menuItems {
		if item.name == name {
			return item.display
		}
	}
	return name
}

func canonicalMenuItem(name string) string {
	for _, item := range menuItems {
		if item.display == name {
			return item.name
		}
	}
	return name
}

// MenuEntry is shared display metadata for the command line and settings UI.
type MenuEntry struct {
	Key, Name, Group string
	Parts            []string
}

func MenuEntries() []MenuEntry {
	return Default().OrderedMenuEntries()
}

// OrderedMenuEntries returns display metadata in the user's saved group, entry,
// and component order.
func (c Config) OrderedMenuEntries() []MenuEntry {
	layout := completeMenuLayout(c.MenuLayout)
	entries := make([]MenuEntry, 0, len(menuItems))
	for _, group := range layout.Groups {
		for _, name := range layout.Items[group] {
			entries = append(entries, MenuEntry{name, DisplayMenuItem(name), group, append([]string(nil), layout.Components[name]...)})
		}
	}
	return entries
}

func moveValue(values []string, value string, step int) bool {
	index := slices.Index(values, value)
	target := index + step
	if index < 0 || target < 0 || target >= len(values) {
		return false
	}
	values[index], values[target] = values[target], values[index]
	return true
}

// MoveMenuGroup moves a complete dashboard section. HEADER remains pinned
// because its clock is part of the title rather than a dashboard section.
func (c *Config) MoveMenuGroup(group string, step int) bool {
	if group == "HEADER" || step == 0 {
		return false
	}
	next := c.Clone()
	next.MenuLayout = completeMenuLayout(next.MenuLayout)
	index := slices.Index(next.MenuLayout.Groups, group)
	target := index + step
	if index < 0 || target < 0 || target >= len(next.MenuLayout.Groups) || next.MenuLayout.Groups[target] == "HEADER" {
		return false
	}
	if !moveValue(next.MenuLayout.Groups, group, step) {
		return false
	}
	*c = next
	return true
}

// MoveMenuItem moves an entry inside its current group.
func (c *Config) MoveMenuItem(name string, step int) bool {
	name = canonicalMenuItem(name)
	next := c.Clone()
	next.MenuLayout = completeMenuLayout(next.MenuLayout)
	for group, items := range next.MenuLayout.Items {
		if moveValue(items, name, step) {
			next.MenuLayout.Items[group] = items
			*c = next
			return true
		}
	}
	return false
}

// MoveDisplayPart moves one component inside a single entry.
func (c *Config) MoveDisplayPart(name, part string, step int) bool {
	name = canonicalMenuItem(name)
	next := c.Clone()
	next.MenuLayout = completeMenuLayout(next.MenuLayout)
	parts := next.MenuLayout.Components[name]
	if !moveValue(parts, part, step) {
		return false
	}
	next.MenuLayout.Components[name] = parts
	*c = next
	return true
}

func (c *Config) ResetMenu() {
	c.Visible = defaultVisibility()
	c.MenuLayout = defaultMenuLayout()
}
