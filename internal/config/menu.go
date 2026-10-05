package config

import "fmt"

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
	{"network-status", "status", "NETWORK", true, true},
	{"network-traffic", "traffic", "NETWORK", true, true},
	{"ip", "ip", "NETWORK", false, false},
	{"cpu", "cpu", "RESOURCES", true, true},
	{"cores", "cores", "RESOURCES", false, true},
	{"ram", "ram", "RESOURCES", true, true},
	{"volume", "volume", "RESOURCES", true, true},
	{"battery", "battery", "RESOURCES", true, true},
	{"temperature", "thermal", "RESOURCES", false, false},
	{"directory", "path", "SESSION", true, false},
	{"spotify", "spotify", "SESSION", true, false},
	{"progress", "progress", "SESSION", true, false},
	{"time", "time", "HEADER", true, false},
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
		return []string{"bar", "percent", "value", "top"}
	case "ram":
		return []string{"bar", "percent", "value", "pressure"}
	case "battery":
		return []string{"bar", "percent", "value", "remaining"}
	case "network-traffic":
		return []string{"upload", "download", "latency"}
	case "network-status":
		return []string{"value", "interface", "ip", "connection", "name"}
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
		"bar": "Bar", "percent": "Percent", "value": "Value", "top": "Top process",
		"pressure": "Pressure", "remaining": "Remaining", "interface": "Interface",
		"ip": "IP", "connection": "Connection", "name": "Network name",
		"timezone": "Timezone", "utc": "UTC offset",
		"upload": "Upload", "download": "Download", "latency": "Latency",
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
	case "cpu.top", "ram.pressure", "battery.bar", "battery.remaining", "network-status.connection", "network-status.name", "date.timezone", "date.utc", "network-traffic.latency":
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
	entries := make([]MenuEntry, 0, len(menuItems))
	for _, item := range menuItems {
		entries = append(entries, MenuEntry{item.name, item.display, item.group, DisplayParts(item.name)})
	}
	return entries
}

func (c *Config) ResetMenu() { c.Visible = defaultVisibility() }
