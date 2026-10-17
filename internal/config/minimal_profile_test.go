package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestMinimalAutomaticPriorityAndManualSelection(t *testing.T) {
	project := t.TempDir()
	if err := os.WriteFile(filepath.Join(project, "go.mod"), []byte("module example.test/project\n"), 0600); err != nil {
		t.Fatal(err)
	}
	c := Default()
	c.Style = "aurora"
	c.Background = "#123456"
	if err := c.SetProfileSelection("auto"); err != nil {
		t.Fatal(err)
	}
	before := c.Clone()
	for _, size := range [][2]int{{109, 40}, {140, 17}} {
		if got := c.ActiveProfile(project, size[0], size[1]); got != "minimal" {
			t.Fatalf("size %v selected %q", size, got)
		}
	}
	if got := c.ActiveProfile(project, 110, 18); got != "coding" {
		t.Fatalf("large project selected %q", got)
	}
	minimal := c.EffectiveProfile(project, 70, 15)
	if minimal.RuntimeProfile != "minimal" || minimal.Style != c.Style || minimal.Animation != c.Animation || minimal.Background != c.Background {
		t.Fatal("Minimal changed the chosen theme")
	}
	for _, key := range []string{"directory", "time", "cpu.percent", "ram.percent"} {
		if !Shown(minimal.Visible, key) {
			t.Fatalf("Minimal hid %s", key)
		}
	}
	for _, key := range []string{"spotify", "cpu.bar", "ram.bar", "runtimes"} {
		if Shown(minimal.Visible, key) {
			t.Fatalf("Minimal showed %s", key)
		}
	}
	if !reflect.DeepEqual(c, before) {
		t.Fatal("Minimal changed the saved configuration")
	}
	if err := c.SelectProfile("minimal"); err != nil {
		t.Fatal(err)
	}
	if got := c.ActiveProfile(t.TempDir(), 140, 40); got != "minimal" {
		t.Fatalf("manual Minimal selected %q", got)
	}
}
