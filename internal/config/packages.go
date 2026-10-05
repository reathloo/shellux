package config

import (
	"context"
	"sort"

	"github.com/reathloo/shellux/internal/themepack"
)

// RequiredPackages includes only the active combination, not stored presets.
func (c Config) RequiredPackages() []string {
	names := map[string]bool{}
	for _, name := range []string{c.Animation, c.Style} {
		if _, ok := themepack.Lookup(name); ok {
			names[name] = true
		}
	}
	base := c.Background == "default" || c.Background == ""
	for _, color := range backgroundColors {
		if color == c.Background {
			base = true
		}
	}
	if !base {
		for _, e := range themepack.Entries() {
			if e.Background == c.Background {
				names[e.Name] = true
				break
			}
		}
	}
	result := []string{}
	for name := range names {
		result = append(result, name)
	}
	sort.Strings(result)
	return result
}

func (c Config) EnsurePackages(ctx context.Context) error {
	return themepack.EnsureNames(ctx, c.RequiredPackages())
}
