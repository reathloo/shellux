package config

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// ProfileNames contains the available preset layouts.
var ProfileNames = []string{"coding", "minimal", "remote", "presentation", "focus", "snapshot"}

var codingMarkers = []string{"go.mod", "package.json", "pyproject.toml", "Cargo.toml", "pom.xml", "build.gradle", "build.gradle.kts"}

func ValidProfile(name string) bool {
	for _, profile := range ProfileNames {
		if name == profile {
			return true
		}
	}
	return false
}

// SelectProfile records a manual choice. An empty name uses standard settings.
func (c *Config) SelectProfile(name string) error {
	next := c.Clone()
	if name != "focus" {
		next.FocusUntil = ""
		next.FocusMinutes = 0
		next.FocusPreviousSelection = ""
		next.FocusPreviousProfile = ""
	} else if c.SelectedProfile != "focus" || c.ProfileSelection != "manual" {
		next.FocusPreviousSelection = c.ProfileSelection
		next.FocusPreviousProfile = c.SelectedProfile
		next.FocusMinutes = 30
		next.FocusUntil = ""
	}
	next.ProfileSelection = "manual"
	next.SelectedProfile = name
	if err := next.Validate(); err != nil {
		return err
	}
	*c = next
	return nil
}

// SetProfileSelection switches between manual choice and automatic detection.
func (c *Config) SetProfileSelection(selection string) error {
	next := c.Clone()
	next.ProfileSelection = selection
	if selection == "auto" {
		if next.SelectedProfile == "focus" {
			next.SelectedProfile = next.FocusPreviousProfile
		}
		next.FocusUntil = ""
		next.FocusMinutes = 0
		next.FocusPreviousSelection = ""
		next.FocusPreviousProfile = ""
	}
	if err := next.Validate(); err != nil {
		return err
	}
	*c = next
	return nil
}

// ActiveProfile resolves the one profile applicable to this directory.
// Remote stays a placeholder until its behavior is defined.
const minimalColumns, minimalRows = 110, 18

func (c Config) ActiveProfile(directory string, size ...int) string {
	if c.ProfileSelection == "manual" && c.SelectedProfile == "focus" && c.FocusUntil != "" {
		until, _ := time.Parse(time.RFC3339Nano, c.FocusUntil)
		if !time.Now().Before(until) {
			previous := c
			previous.ProfileSelection = c.FocusPreviousSelection
			previous.SelectedProfile = c.FocusPreviousProfile
			if previous.ProfileSelection == "" {
				previous.ProfileSelection = "manual"
			}
			if previous.SelectedProfile == "focus" {
				previous.SelectedProfile = ""
			}
			return previous.ActiveProfile(directory, size...)
		}
	}
	if c.ProfileSelection == "auto" {
		if len(size) >= 2 && size[0] > 0 && size[1] > 0 && (size[0] < minimalColumns || size[1] < minimalRows) {
			return "minimal"
		}
		if isCodingProject(directory) {
			return "coding"
		}
		return ""
	}
	if c.SelectedProfile == "coding" || c.SelectedProfile == "minimal" || c.SelectedProfile == "presentation" || c.SelectedProfile == "snapshot" || c.SelectedProfile == "focus" {
		return c.SelectedProfile
	}
	return ""
}

// SetFocusDuration prepares a timed session. ActivateFocus starts it on save.
func (c *Config) SetFocusDuration(minutes int) error {
	if minutes < 1 || minutes > 240 {
		return fmt.Errorf("focus timer must be 1 to 240 minutes")
	}
	if err := c.SelectProfile("focus"); err != nil {
		return err
	}
	c.FocusMinutes = minutes
	c.FocusUntil = ""
	return nil
}

func (c *Config) ActivateFocus(now time.Time) {
	if c.ProfileSelection == "manual" && c.SelectedProfile == "focus" && c.FocusUntil == "" && c.FocusMinutes > 0 {
		c.FocusUntil = now.Add(time.Duration(c.FocusMinutes) * time.Minute).Format(time.RFC3339Nano)
	}
}

func (c Config) FocusRemaining(now time.Time) time.Duration {
	until, err := time.Parse(time.RFC3339Nano, c.FocusUntil)
	if err != nil || !now.Before(until) || c.ProfileSelection != "manual" || c.SelectedProfile != "focus" {
		return 0
	}
	return until.Sub(now)
}

// RestoreExpiredFocus normalizes a finished session when loading settings.
func (c *Config) RestoreExpiredFocus(now time.Time) {
	if c.SelectedProfile != "focus" || c.FocusUntil == "" {
		return
	}
	until, err := time.Parse(time.RFC3339Nano, c.FocusUntil)
	if err != nil || now.Before(until) {
		return
	}
	selection, profile := c.FocusPreviousSelection, c.FocusPreviousProfile
	if selection == "" {
		selection = "manual"
	}
	c.ProfileSelection, c.SelectedProfile = selection, profile
	c.FocusUntil, c.FocusMinutes, c.FocusPreviousSelection, c.FocusPreviousProfile = "", 0, "", ""
}

func isCodingProject(directory string) bool {
	return hasProjectMarker(directory, append([]string{".git"}, codingMarkers...))
}

func hasProjectMarker(directory string, markers []string) bool {
	directory, err := filepath.Abs(directory)
	if err != nil {
		return false
	}
	home, _ := os.UserHomeDir()
	start := directory
	for {
		if directory != home || start == home {
			for _, marker := range markers {
				if _, err := os.Stat(filepath.Join(directory, marker)); err == nil {
					return true
				}
			}
		}
		parent := filepath.Dir(directory)
		if parent == directory || directory == home {
			return false
		}
		directory = parent
	}
}

// EffectiveProfile returns a temporary preset menu without changing saved settings.
func (c Config) EffectiveProfile(directory string, size ...int) Config {
	profile := c.ActiveProfile(directory, size...)
	if profile != "coding" && profile != "minimal" && profile != "presentation" && profile != "snapshot" && profile != "focus" {
		return c
	}
	effective := c.Clone()
	effective.RuntimeProfile = profile
	if profile == "focus" {
		effective.Background = "#000000"
	}
	for _, item := range menuItems {
		effective.Visible[item.name] = false
		for _, part := range DisplayParts(item.name) {
			effective.Visible[item.name+"."+part] = false
		}
	}
	if profile == "presentation" {
		for _, name := range []string{"time", "date", "uptime", "cpu", "ram", "network-status"} {
			effective.Visible[name] = true
		}
		for _, name := range []string{"date.value", "cpu.percent", "ram.percent", "network-status.value"} {
			effective.Visible[name] = true
		}
		effective.MenuLayout = defaultMenuLayout()
		effective.MenuLayout.Groups = []string{"SYSTEM", "RESOURCES", "NETWORK", "HEADER", "CODING", "SESSION"}
		return effective
	}
	if profile == "snapshot" {
		for _, name := range []string{"date", "platform", "shell", "uptime", "cpu", "ram", "load-average", "network-status"} {
			effective.Visible[name] = true
		}
		for _, name := range []string{"date.value", "cpu.value", "cpu.percent", "ram.value", "ram.percent", "load-average.one", "load-average.five", "load-average.fifteen", "network-status.value"} {
			effective.Visible[name] = true
		}
		if os.Getenv("SHELLUX_LAST_COMMAND_STATUS") != "" && os.Getenv("SHELLUX_LAST_COMMAND_SECONDS") != "" {
			effective.Visible["last-command"] = true
			effective.Visible["last-command.duration"] = true
			effective.Visible["last-command.exit"] = true
		}
		effective.MenuLayout = defaultMenuLayout()
		effective.MenuLayout.Groups = []string{"SYSTEM", "RESOURCES", "NETWORK", "CODING", "SESSION", "HEADER"}
		return effective
	}
	if profile == "focus" {
		for _, name := range []string{"time", "cpu", "ram", "spotify", "progress"} {
			effective.Visible[name] = true
		}
		for _, name := range []string{"cpu.percent", "ram.percent"} {
			effective.Visible[name] = true
		}
		effective.MenuLayout = defaultMenuLayout()
		effective.MenuLayout.Groups = []string{"RESOURCES", "SYSTEM", "NETWORK", "CODING", "SESSION", "HEADER"}
		return effective
	}
	for _, name := range []string{"directory", "cpu", "ram", "time"} {
		effective.Visible[name] = true
	}
	if profile == "minimal" {
		effective.Visible["cpu.percent"] = true
		effective.Visible["ram.percent"] = true
		if hasProjectMarker(directory, []string{".git"}) {
			effective.Visible["git-branch"] = true
			effective.Visible["git-branch.branch"] = true
		}
		effective.MenuLayout = defaultMenuLayout()
		effective.MenuLayout.Groups = []string{"CODING", "SESSION", "RESOURCES", "HEADER", "SYSTEM", "NETWORK"}
		return effective
	}
	if hasProjectMarker(directory, []string{".git"}) {
		effective.Visible["git-branch"] = true
		effective.Visible["git-sync"] = true
	}
	for name, parts := range map[string][]string{
		"git-branch": {"branch", "repository"},
		"git-sync":   {"ahead", "behind"},
		"cpu":        {"bar", "percent"},
		"ram":        {"bar", "percent"},
	} {
		for _, part := range parts {
			effective.Visible[name+"."+part] = true
		}
	}
	for tool, part := range map[string]string{"go": "go", "node": "node", "python3": "python", "java": "java"} {
		if _, err := exec.LookPath(tool); err == nil {
			effective.Visible["runtimes"] = true
			effective.Visible["runtimes."+part] = true
		}
	}
	for tool, part := range map[string]string{"docker": "docker", "kubectl": "kubernetes"} {
		if _, err := exec.LookPath(tool); err == nil {
			effective.Visible["dev-context"] = true
			effective.Visible["dev-context."+part] = true
		}
	}
	if os.Getenv("SHELLUX_LAST_COMMAND_STATUS") != "" && os.Getenv("SHELLUX_LAST_COMMAND_SECONDS") != "" {
		effective.Visible["last-command"] = true
		effective.Visible["last-command.duration"] = true
		effective.Visible["last-command.exit"] = true
	}
	effective.MenuLayout = defaultMenuLayout()
	effective.MenuLayout.Groups = []string{"CODING", "SESSION", "RESOURCES", "NETWORK", "SYSTEM", "HEADER"}
	return effective
}
