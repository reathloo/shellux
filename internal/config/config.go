package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/reathloo/shellux/internal/locale"
	"github.com/reathloo/shellux/internal/themepack"
)

var baseAnimations = []string{"shelluxdefault"}

// styles contains the public command names. luxury is stored as dark-luxury
// for compatibility with existing configuration files.
var baseStyles = []string{"luxury", "neon", "aurora", "rainbow", "shelluxdefault"}

// SavedTheme combines an animation, dashboard style and terminal background.
type SavedTheme struct {
	Background string `json:"background"`
	Animation  string `json:"animation"`
	Style      string `json:"style"`
}

// ThemeInfo is a named theme suitable for displaying in the command line.
type ThemeInfo struct {
	Background string
	Name       string
	Animation  string
	Style      string
	Custom     bool
}

func availableThemes() map[string]SavedTheme {
	result := map[string]SavedTheme{"shelluxdefault": {Animation: "shelluxdefault", Style: "shelluxdefault", Background: "#000000"}}
	for _, e := range themepack.Entries() {
		result[e.Name] = SavedTheme{Animation: e.Name, Style: e.Name, Background: e.Background}
	}
	return result
}

// Config contains the user-facing Shellux settings.
type Config struct {
	Language        string                `json:"language,omitempty"`
	Background      string                `json:"background,omitempty"`
	Animation       string                `json:"animation"`
	Style           string                `json:"style"`
	RefreshInterval string                `json:"refresh_interval"`
	Visible         map[string]bool       `json:"visible"`
	Themes          map[string]SavedTheme `json:"themes,omitempty"`
}

func Default() Config {
	base := availableThemes()["shelluxdefault"]
	return Config{
		Language:        "auto",
		Background:      base.Background,
		Animation:       base.Animation,
		Style:           base.Style,
		RefreshInterval: "1s",
		Visible:         defaultVisibility(),
	}
}

func Load(path string) (Config, error) {
	if path == "" {
		return Default(), nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}
	return decodeConfig(data)
}

func decodeConfig(data []byte) (Config, error) {
	config := Default()
	if err := json.Unmarshal(data, &config); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}
	var legacy struct {
		Background *string         `json:"background"`
		Style      *string         `json:"style"`
		Theme      *string         `json:"theme"`
		Visible    map[string]bool `json:"visible"`
	}
	if err := json.Unmarshal(data, &legacy); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}
	migrateThemeNames(&config)
	// Older active configurations and saved themes retain their profile background.
	if legacy.Background == nil || config.Background == "" {
		config.Background = "default"
	}
	background, err := BackgroundColor(config.Background)
	if err != nil {
		return Config{}, err
	}
	config.Background = background
	for name, saved := range config.Themes {
		if saved.Background == "" {
			saved.Background = "default"
		}
		saved.Background, err = BackgroundColor(saved.Background)
		if err != nil {
			return Config{}, fmt.Errorf("theme %q: %w", name, err)
		}
		config.Themes[name] = saved
	}
	if legacy.Style == nil && legacy.Theme != nil {
		config.Style = canonicalStyle(*legacy.Theme)
	}
	if oldVisible, exists := legacy.Visible["playback"]; exists {
		for _, name := range []string{"spotify", "progress"} {
			if _, configured := legacy.Visible[name]; !configured {
				config.Visible[name] = oldVisible
			}
		}
		delete(config.Visible, "playback")
	}
	if config.Visible == nil {
		config.Visible = Default().Visible
	} else {
		for name, visible := range Default().Visible {
			if _, configured := config.Visible[name]; !configured {
				config.Visible[name] = visible
			}
		}
	}
	migrateTraffic(config.Visible, legacy.Visible)
	if err := config.Validate(); err != nil {
		return Config{}, err
	}
	return config, nil
}

// LoadDefault loads the optional user configuration. A missing default file is
// equivalent to using the built-in defaults.
func LoadDefault() (Config, error) {
	path, err := DefaultPath()
	if err != nil {
		return Config{}, err
	}
	config, err := Load(path)
	if errors.Is(err, os.ErrNotExist) {
		return Default(), nil
	}
	return config, err
}

// DefaultPath returns the user-level Shellux configuration location.
func DefaultPath() (string, error) {
	directory := os.Getenv("XDG_CONFIG_HOME")
	if directory == "" {
		var err error
		directory, err = os.UserConfigDir()
		if err != nil {
			return "", fmt.Errorf("find user config directory: %w", err)
		}
	}
	return filepath.Join(directory, "shellux", "config.json"), nil
}

// SetAnimation persists the selected header animation in the default user config.
func SetAnimation(name string) (string, error) {
	settings, err := LoadDefault()
	if err != nil {
		return "", err
	}
	if err := settings.ChangeAppearance("animation", name); err != nil {
		return "", err
	}
	return writeDefault(settings)
}

// SetStyle persists the selected visual style in the default user config.
func SetStyle(name string) (string, error) {
	settings, err := LoadDefault()
	if err != nil {
		return "", err
	}
	if err := settings.ChangeAppearance("style", name); err != nil {
		return "", err
	}
	return writeDefault(settings)
}

// Animations returns every built-in animation name.
func Animations() []string {
	result := append([]string(nil), baseAnimations...)
	for _, e := range themepack.Entries() {
		result = append(result, e.Name)
	}
	return result
}

// Styles returns every public style name.
func Styles() []string {
	result := append([]string(nil), baseStyles...)
	for _, e := range themepack.Entries() {
		result = append(result, e.Name)
	}
	return result
}

// Themes returns built-in and user-created themes. The default theme is listed
// first, followed by built-ins and alphabetically ordered custom themes.
func Themes() ([]ThemeInfo, error) {
	settings, err := LoadDefault()
	if err != nil {
		return nil, err
	}
	return settings.ThemeOptions(), nil
}

// ThemeOptions lists choices from this working configuration without disk access.
func (c Config) ThemeOptions() []ThemeInfo {
	defaults := Default()
	result := []ThemeInfo{
		{Name: "default", Animation: defaults.Animation, Style: DisplayStyle(defaults.Style), Background: defaults.Background},
	}
	builtInNames := make([]string, 0, len(availableThemes()))
	for name := range availableThemes() {
		builtInNames = append(builtInNames, name)
	}
	sort.Strings(builtInNames)
	for _, name := range builtInNames {
		entry := availableThemes()[name]
		result = append(result, ThemeInfo{Name: name, Animation: entry.Animation, Style: DisplayStyle(entry.Style), Background: entry.Background})
	}
	customNames := make([]string, 0, len(c.Themes))
	for name := range c.Themes {
		customNames = append(customNames, name)
	}
	sort.Strings(customNames)
	for _, name := range customNames {
		entry := c.Themes[name]
		result = append(result, ThemeInfo{Name: name, Animation: entry.Animation, Style: DisplayStyle(entry.Style), Background: entry.Background, Custom: true})
	}
	return result
}

// SetTheme applies a built-in, custom, default, or random theme.
func SetTheme(name string) (SavedTheme, string, error) {
	settings, err := LoadDefault()
	if err != nil {
		return SavedTheme{}, "", err
	}
	selected, err := settings.ApplyTheme(name)
	if err != nil {
		return SavedTheme{}, "", err
	}
	path, err := writeDefault(settings)
	return selected, path, err
}

// ApplyTheme changes the draft and uses the same rules as the CLI.
func (c *Config) ApplyTheme(name string) (SavedTheme, error) {
	name = canonicalThemeName(name)
	var selected SavedTheme
	switch name {
	case "default":
		defaults := Default()
		selected = SavedTheme{Animation: defaults.Animation, Style: defaults.Style, Background: defaults.Background}
	case "random":
		animations := Animations()
		styles := Styles()
		selected = availableThemes()[animations[rand.IntN(len(animations))]]
		selected.Style = canonicalStyle(styles[rand.IntN(len(styles))])
	default:
		var found bool
		selected, found = availableThemes()[name]
		if !found {
			selected, found = c.Themes[name]
		}
		if !found {
			return SavedTheme{}, fmt.Errorf("unknown theme %q", name)
		}
	}
	next := c.Clone()
	next.Animation = selected.Animation
	next.Style = canonicalStyle(selected.Style)
	next.Background = selected.Background
	if err := next.Validate(); err != nil {
		return SavedTheme{}, err
	}
	*c = next
	return SavedTheme{Animation: c.Animation, Style: c.Style, Background: c.Background}, nil
}

// SaveCurrentTheme stores the selected animation, style and background under a
// user-provided name.
func SaveCurrentTheme(name string) (string, string, error) {
	settings, err := LoadDefault()
	if err != nil {
		return "", "", err
	}
	name, err = settings.StoreTheme(name)
	if err != nil {
		return "", "", err
	}
	path, err := writeDefault(settings)
	return name, path, err
}

// StoreTheme stores the current combination in the working copy.
func (c *Config) StoreTheme(name string) (string, error) {
	name, err := customThemeName(name)
	if err != nil {
		return "", err
	}
	next := c.Clone()
	if next.Themes == nil {
		next.Themes = make(map[string]SavedTheme)
	}
	next.Themes[name] = SavedTheme{Animation: c.Animation, Style: c.Style, Background: c.Background}
	if err := next.Validate(); err != nil {
		return "", err
	}
	*c = next
	return name, nil
}

// DeleteTheme removes a user-created theme. Built-in themes cannot be deleted.
func DeleteTheme(name string) (string, string, error) {
	settings, err := LoadDefault()
	if err != nil {
		return "", "", err
	}
	name, err = settings.RemoveTheme(name)
	if err != nil {
		return "", "", err
	}
	path, err := writeDefault(settings)
	return name, path, err
}

// RemoveTheme protects built-ins, including in an unsaved working copy.
func (c *Config) RemoveTheme(name string) (string, error) {
	name = canonicalThemeName(name)
	if name == "default" || name == "random" || name == "new" || name == "delete" {
		return "", fmt.Errorf("theme %q is built in and cannot be deleted", name)
	}
	if _, builtIn := availableThemes()[name]; builtIn {
		return "", fmt.Errorf("theme %q is built in and cannot be deleted", name)
	}
	if _, exists := c.Themes[name]; !exists {
		return "", fmt.Errorf("custom theme %q does not exist", name)
	}
	delete(c.Themes, name)
	if len(c.Themes) == 0 {
		c.Themes = nil
	}
	return name, nil
}

func writeDefault(settings Config) (string, error) {
	path, err := DefaultPath()
	if err != nil {
		return "", err
	}
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return "", fmt.Errorf("encode config: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return "", fmt.Errorf("create config directory: %w", err)
	}
	// Preserve symlinked dotfiles, but replace the destination atomically so
	// another terminal can never read a truncated or partially written config.
	target := path
	if _, err := os.Lstat(path); err == nil {
		target, err = filepath.EvalSymlinks(path)
		if err != nil {
			return "", fmt.Errorf("resolve config path: %w", err)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("resolve config path: %w", err)
	}
	temporary, err := os.CreateTemp(filepath.Dir(target), ".shellux-config-*")
	if err != nil {
		return "", fmt.Errorf("create temporary config: %w", err)
	}
	defer os.Remove(temporary.Name())
	_, writeErr := temporary.Write(append(data, '\n'))
	closeErr := temporary.Close()
	if err := errors.Join(writeErr, closeErr); err != nil {
		return "", fmt.Errorf("write config: %w", err)
	}
	if err := os.Rename(temporary.Name(), target); err != nil {
		return "", fmt.Errorf("replace config: %w", err)
	}
	return path, nil
}

func (c Config) Validate() error {
	if !locale.Valid(c.Language) {
		return fmt.Errorf("unsupported language %q", c.Language)
	}
	if err := validateMenu(c.Visible); err != nil {
		return err
	}
	if c.Background != "" {
		if _, err := BackgroundColor(c.Background); err != nil {
			return err
		}
	}
	if !contains(Animations(), c.Animation) {
		return fmt.Errorf("unsupported animation %q", c.Animation)
	}
	if !contains(Styles(), DisplayStyle(c.Style)) {
		return fmt.Errorf("unsupported style %q", c.Style)
	}
	interval, err := time.ParseDuration(c.RefreshInterval)
	if err != nil || interval <= 0 {
		return fmt.Errorf("invalid refresh interval %q", c.RefreshInterval)
	}
	for name, saved := range c.Themes {
		if _, err := customThemeName(name); err != nil {
			return err
		}
		if !contains(Animations(), saved.Animation) {
			return fmt.Errorf("theme %q has unsupported animation %q", name, saved.Animation)
		}
		if !contains(Styles(), DisplayStyle(saved.Style)) {
			return fmt.Errorf("theme %q has unsupported style %q", name, saved.Style)
		}
		if _, err := BackgroundColor(saved.Background); err != nil {
			return fmt.Errorf("theme %q: %w", name, err)
		}
	}
	return nil
}

func canonicalStyle(name string) string {
	name = canonicalThemeName(name)
	if name == "luxury" {
		return "dark-luxury"
	}
	return name
}

// DisplayStyle returns the public name for a persisted style.
func DisplayStyle(name string) string {
	if name == "dark-luxury" {
		return "luxury"
	}
	return canonicalThemeName(name)
}

func customThemeName(name string) (string, error) {
	name = canonicalThemeName(name)
	if name == "" || len(name) > 32 {
		return "", errors.New("theme name must be 1 to 32 letters, numbers, or hyphens")
	}
	if name == "default" || name == "random" || name == "new" || name == "delete" {
		return "", fmt.Errorf("theme name %q is reserved", name)
	}
	if _, builtIn := availableThemes()[name]; builtIn {
		return "", fmt.Errorf("theme name %q is built in", name)
	}
	for index, character := range name {
		if (character < 'a' || character > 'z') && (character < '0' || character > '9') && character != '-' {
			return "", errors.New("theme name must be 1 to 32 letters, numbers, or hyphens")
		}
		if character == '-' && (index == 0 || index == len(name)-1) {
			return "", errors.New("theme name cannot start or end with a hyphen")
		}
	}
	return name, nil
}

func normalizeName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func canonicalThemeName(name string) string {
	return normalizeName(name)
}

func migrateThemeNames(config *Config) {
	config.Animation = canonicalThemeName(config.Animation)
	config.Style = canonicalStyle(config.Style)
	for name, saved := range config.Themes {
		saved.Animation = canonicalThemeName(saved.Animation)
		saved.Style = canonicalStyle(saved.Style)
		config.Themes[name] = saved
	}
}

func contains(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

func (c Config) Interval() time.Duration {
	interval, _ := time.ParseDuration(c.RefreshInterval)
	return interval
}
