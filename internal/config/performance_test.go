package config

import (
	"fmt"
	"testing"
)

func BenchmarkValidateThemes(b *testing.B) {
	c := Default()
	c.Themes = make(map[string]SavedTheme, 20)
	c.FavoriteThemes = make(map[string]bool, 20)
	for i := range 20 {
		name := fmt.Sprintf("custom-%d", i)
		c.Themes[name] = SavedTheme{Animation: "shelluxdefault", Style: "dark-luxury", Background: "#000000"}
		c.FavoriteThemes[name] = true
	}
	b.ReportAllocs()
	for b.Loop() {
		if err := c.Validate(); err != nil {
			b.Fatal(err)
		}
	}
}
