package config

import (
	"errors"
	"fmt"
	"maps"
	"reflect"
	"sort"
)

// SavedMenuPreset captures only menu visibility and ordering. Theme and general
// settings remain independent.
type SavedMenuPreset struct {
	Visible map[string]bool `json:"visible"`
	Layout  MenuLayout      `json:"layout"`
}

func cloneSavedMenuPreset(preset SavedMenuPreset) SavedMenuPreset {
	preset.Visible = maps.Clone(preset.Visible)
	preset.Layout.Groups = append([]string(nil), preset.Layout.Groups...)
	preset.Layout.Items = cloneStringSlices(preset.Layout.Items)
	preset.Layout.Components = cloneStringSlices(preset.Layout.Components)
	return preset
}

func completeMenuPresetVisibility(visible map[string]bool) map[string]bool {
	result := defaultVisibility()
	for name, shown := range visible {
		result[name] = shown
	}
	return result
}

func customMenuPresetName(name string) (string, error) {
	name = normalizeName(name)
	if name == "" || len(name) > 32 {
		return "", errors.New("menu preset name must have 1 to 32 characters")
	}
	if name == "default" || name == "new" || name == "delete" {
		return "", fmt.Errorf("menu preset name %q is reserved", name)
	}
	for index, character := range name {
		if (character < 'a' || character > 'z') && (character < '0' || character > '9') && character != '-' {
			return "", errors.New("menu preset name must use letters a-z, digits or hyphens")
		}
		if character == '-' && (index == 0 || index == len(name)-1) {
			return "", errors.New("menu preset name cannot start or end with a hyphen")
		}
	}
	return name, nil
}

// MenuPresetOptions lists the built-in default and saved layouts by name.
func (c Config) MenuPresetOptions() []string {
	names := []string{"default"}
	for name := range c.MenuPresets {
		names = append(names, name)
	}
	sort.Strings(names[1:])
	return names
}

// MenuPresetMatches reports whether a preset represents the current menu.
func (c Config) MenuPresetMatches(name string) bool {
	var preset SavedMenuPreset
	if name == "default" {
		base := Default()
		preset = SavedMenuPreset{Visible: base.Visible, Layout: base.MenuLayout}
	} else {
		var exists bool
		preset, exists = c.MenuPresets[name]
		if !exists {
			return false
		}
	}
	return maps.Equal(c.Visible, preset.Visible) && reflect.DeepEqual(completeMenuLayout(c.MenuLayout), completeMenuLayout(preset.Layout))
}

// StoreMenuPreset saves the current menu in the working configuration draft.
func (c *Config) StoreMenuPreset(name string) (string, error) {
	name, err := customMenuPresetName(name)
	if err != nil {
		return "", err
	}
	next := c.Clone()
	if next.MenuPresets == nil {
		next.MenuPresets = make(map[string]SavedMenuPreset)
	}
	next.MenuPresets[name] = SavedMenuPreset{Visible: maps.Clone(c.Visible), Layout: completeMenuLayout(c.MenuLayout)}
	if err := next.Validate(); err != nil {
		return "", err
	}
	*c = next
	return name, nil
}

// ApplyMenuPreset replaces only the active menu choices and order.
func (c *Config) ApplyMenuPreset(name string) error {
	name = normalizeName(name)
	next := c.Clone()
	if name == "default" {
		next.ResetMenu()
	} else {
		preset, exists := next.MenuPresets[name]
		if !exists {
			return fmt.Errorf("unknown menu preset %q", name)
		}
		selected := cloneSavedMenuPreset(preset)
		next.Visible = selected.Visible
		next.MenuLayout = selected.Layout
	}
	if err := next.Validate(); err != nil {
		return err
	}
	*c = next
	return nil
}

// RemoveMenuPreset deletes a saved layout without changing the active menu.
func (c *Config) RemoveMenuPreset(name string) error {
	name = normalizeName(name)
	if name == "default" {
		return errors.New("default menu preset cannot be deleted")
	}
	if _, exists := c.MenuPresets[name]; !exists {
		return fmt.Errorf("unknown menu preset %q", name)
	}
	next := c.Clone()
	delete(next.MenuPresets, name)
	if len(next.MenuPresets) == 0 {
		next.MenuPresets = nil
	}
	*c = next
	return nil
}
