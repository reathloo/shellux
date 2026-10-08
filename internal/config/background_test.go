package config

import (
	"os"
	"testing"
)

func TestBackgroundColor(t *testing.T) {
	for _, test := range []struct{ input, want string }{
		{"dark", "#11111b"}, {"NAVY", "#000080"}, {" #A0B1C2 ", "#a0b1c2"}, {"default", "default"}, {"bluerunner", "#080b1a"}, {"purplemonster", "#140d20"}, {"orangedragon", "#1e1210"}, {"orb", "#0b1420"}, {"liqudemetall", "#11131c"}, {"anime-face", "#100317"}, {"loopingliqude", "#17191f"}, {"heart", "#1b0d15"}, {"pinkcat", "#24121d"}, {"sleepyguy", "#120f12"}, {"spiderboy", "#0c1018"}, {"shelluxdefault", "#000000"}, {"slotmaschine", "#100d12"},
	} {
		got, err := BackgroundColor(test.input)
		if err != nil || got != test.want {
			t.Fatalf("BackgroundColor(%q) = %q, %v", test.input, got, err)
		}
	}
	for _, invalid := range []string{"", "unknown", "#123", "#gg0000", "#112233;red", "#112233\033]0;injected\007"} {
		if _, err := BackgroundColor(invalid); err == nil {
			t.Fatalf("accepted invalid color %q", invalid)
		}
	}
}

func TestBackgroundPersistsIndependentlyAndRejectsInvalidInput(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if _, err := SetStyle("neon"); err != nil {
		t.Fatal(err)
	}
	color, path, err := SetBackground("navy")
	if err != nil || color != "#000080" {
		t.Fatalf("SetBackground() = %q, %v", color, err)
	}
	settings, err := LoadDefault()
	if err != nil || settings.Background != color || settings.Style != "neon" {
		t.Fatalf("background lost other settings: %+v %v", settings, err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := SetBackground("not-a-color"); err == nil {
		t.Fatal("accepted invalid color")
	}
	after, err := os.ReadFile(path)
	if err != nil || string(before) != string(after) {
		t.Fatal("invalid color changed config")
	}
	if _, err := UpdateMenu("reset", nil); err != nil {
		t.Fatal(err)
	}
	settings, err = LoadDefault()
	if err != nil || settings.Background != color {
		t.Fatal("menu reset changed background")
	}
	if _, _, err := SetBackground("default"); err != nil {
		t.Fatal(err)
	}
	settings, err = LoadDefault()
	if err != nil || settings.Background != "default" {
		t.Fatal("background reset was not saved")
	}
	settings.Background = "\033]11;red\007"
	if settings.Validate() == nil {
		t.Fatal("configuration permits injected control sequence")
	}
}
