package config

import (
	"bytes"
	"errors"
	"fmt"
	"maps"
	"os"
	"sort"
)

// Clone owns its maps; cancelling an editor must never mutate its baseline.
func (c Config) Clone() Config {
	c.Visible = maps.Clone(c.Visible)
	if c.Visible == nil {
		c.Visible = defaultVisibility()
	}
	c.Themes = maps.Clone(c.Themes)
	return c
}

// ChangeAppearance validates a change before replacing the draft.
func (c *Config) ChangeAppearance(field, value string) error {
	next := c.Clone()
	switch field {
	case "language":
		next.Language = normalizeName(value)
	case "animation":
		next.Animation = canonicalThemeName(value)
	case "style":
		next.Style = canonicalStyle(value)
	case "background":
		color, err := BackgroundColor(value)
		if err != nil {
			return err
		}
		next.Background = color
	case "interval":
		next.RefreshInterval = value
	default:
		return fmt.Errorf("unknown setting %q", field)
	}
	if err := next.Validate(); err != nil {
		return err
	}
	*c = next
	return nil
}

type BackgroundOption struct{ Name, Color string }

func (c Config) BackgroundOptions() []BackgroundOption {
	result := []BackgroundOption{{"Profile color", "default"}}
	names := make([]string, 0, len(backgroundColors))
	for name := range backgroundColors {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		result = append(result, BackgroundOption{name, backgroundColors[name]})
	}
	for _, theme := range c.ThemeOptions() {
		result = append(result, BackgroundOption{"Theme: " + theme.Name, theme.Background})
	}
	return result
}

// BackgroundName also recognises user-created themes without changing the JSON.
func (c Config) BackgroundName() string {
	name := DisplayBackground(c.Background)
	if name != c.Background {
		return name
	}
	for _, option := range c.ThemeOptions() {
		if option.Custom && option.Background == c.Background {
			return option.Name
		}
	}
	if c.Background == "default" {
		return "Profile color"
	}
	return name
}

// ErrChanged prevents a draft from replacing a config modified during editing.
var ErrChanged = errors.New("The configuration changed externally. Please cancel and reopen settings.")

// EditSession remembers the exact original file, including whether it existed.
// An invalid external rewrite must also prevent saving, rather than be replaced.
type EditSession struct {
	Config Config
	path   string
	data   []byte
	exists bool
}

func OpenEditor() (*EditSession, error) {
	path, err := DefaultPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	exists := err == nil
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	c := Default()
	if exists {
		c, err = decodeConfig(data)
		if err != nil {
			return nil, err
		}
	}
	return &EditSession{Config: c, path: path, data: data, exists: exists}, nil
}

func (s *EditSession) Save(c Config) error {
	if err := c.Validate(); err != nil {
		return err
	}
	path, err := DefaultPath()
	if err != nil {
		return err
	}
	if path != s.path {
		return ErrChanged
	}
	current, err := os.ReadFile(s.path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if (err == nil) != s.exists || !bytes.Equal(current, s.data) {
		return ErrChanged
	}
	_, err = writeDefault(c)
	return err
}
