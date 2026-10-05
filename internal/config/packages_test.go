package config

import (
	"reflect"
	"testing"
)

func TestActivePackageDependencies(t *testing.T) {
	c := Default()
	for _, style := range []string{"shelluxdefault", "luxury", "neon", "aurora", "rainbow"} {
		c.Style = style
		for _, bg := range backgroundColors {
			c.Background = bg
			if len(c.RequiredPackages()) != 0 {
				t.Fatal("base appearance needs download")
			}
		}
	}
	c.Animation = "orb"
	c.Style = "heart"
	c.Background = "#100d12"
	c.Themes = map[string]SavedTheme{"unused": {Animation: "purplemonster", Style: "purplemonster", Background: "#140d20"}}
	if got := c.RequiredPackages(); !reflect.DeepEqual(got, []string{"heart", "orb", "slotmaschine"}) {
		t.Fatalf("dependencies: %v", got)
	}
	if _, err := c.ApplyTheme("heart"); err != nil {
		t.Fatal(err)
	}
	if got := c.RequiredPackages(); !reflect.DeepEqual(got, []string{"heart"}) {
		t.Fatalf("package not deduplicated: %v", got)
	}
}
