package config

import (
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
)

var backgroundColors = map[string]string{
	"black": "#000000", "dark": "#11111b", "gray": "#808080",
	"white": "#ffffff", "red": "#ff0000", "green": "#008000",
	"blue": "#0000ff", "navy": "#000080", "purple": "#800080",
}

// DisplayBackground shows a known background's name, or the custom RGB color.
func DisplayBackground(color string) string {
	color = normalizeName(color)
	for name, theme := range availableThemes() {
		if theme.Background == color {
			return name
		}
	}
	for name, namedColor := range backgroundColors {
		if namedColor == color {
			return name
		}
	}
	return color
}

// BackgroundColor normalizes named colors and validates a six-digit RGB color.
// default restores the terminal profile's background.
func BackgroundColor(name string) (string, error) {
	name = canonicalThemeName(name)
	if name == "default" {
		return name, nil
	}
	if theme, known := availableThemes()[name]; known {
		return theme.Background, nil
	}
	if color, known := backgroundColors[name]; known {
		return color, nil
	}
	if len(name) == 7 && strings.HasPrefix(name, "#") {
		if _, err := hex.DecodeString(name[1:]); err == nil {
			return name, nil
		}
	}
	return "", fmt.Errorf("invalid background %q; use %s, default, or '#RRGGBB'", name, strings.Join(BackgroundNames(), ", "))
}

// SetBackground changes only the background of the current theme combination.
func SetBackground(name string) (string, string, error) {
	settings, err := LoadDefault()
	if err != nil {
		return "", "", err
	}
	if err := settings.ChangeAppearance("background", name); err != nil {
		return "", "", err
	}
	path, err := writeDefault(settings)
	return settings.Background, path, err
}

// BackgroundNames lists accepted names for help and validation messages.
func BackgroundNames() []string {
	names := make([]string, 0, len(backgroundColors)+len(availableThemes()))
	for name := range backgroundColors {
		names = append(names, name)
	}
	for name := range availableThemes() {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
