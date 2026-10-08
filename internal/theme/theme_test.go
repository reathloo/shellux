package theme

import "testing"

func TestByName(t *testing.T) {
	for _, name := range []string{"dark-luxury", "luxury", "neon", "aurora", "rainbow", "bluerunner", "orangedragon", "purplemonster", "orb", "liqudemetall", "anime-face", "loopingliqude", "heart", "pinkcat", "sleepyguy", "spiderboy", "shelluxdefault", "slotmaschine"} {
		style, err := ByName(name)
		if err != nil {
			t.Fatalf("ByName(%q) error = %v", name, err)
		}
		if style.Accent == "" || style.Reset == "" {
			t.Fatalf("ByName(%q) returned incomplete style: %+v", name, style)
		}
	}
	if _, err := ByName("unknown"); err == nil {
		t.Fatal("ByName() accepted an unsupported style")
	}
}

// The shared palette must retain existing terminal colors when UI colors are added.
func TestHeaderColorSequencesRemainCompatible(t *testing.T) {
	for _, test := range []struct{ name, secondary, accent, muted string }{
		{"luxury", "\x1b[90m", "\x1b[38;5;180m", "\x1b[90m"},
		{"neon", "\x1b[38;5;51m", "\x1b[38;5;201m", "\x1b[38;5;45m"},
		{"aurora", "\x1b[38;5;141m", "\x1b[38;5;87m", "\x1b[38;5;103m"},
		{"rainbow", "\x1b[38;5;226m", "\x1b[38;5;201m", "\x1b[38;5;51m"},
		{"bluerunner", "\x1b[38;2;79;79;255m", "\x1b[38;2;255;156;94m", "\x1b[38;2;87;87;148m"},
		{"orangedragon", "\x1b[38;2;255;175;0m", "\x1b[38;2;255;135;0m", "\x1b[38;2;95;135;175m"},
		{"purplemonster", "\x1b[38;2;175;135;255m", "\x1b[38;2;215;135;255m", "\x1b[38;2;95;95;135m"},
		{"orb", "\x1b[38;2;62;53;227m", "\x1b[38;2;188;106;209m", "\x1b[38;2;105;147;146m"},
		{"liqudemetall", "\x1b[38;2;113;110;170m", "\x1b[38;2;217;195;212m", "\x1b[38;2;100;135;151m"},
		{"anime-face", "\x1b[38;2;213;163;183m", "\x1b[38;2;254;245;250m", "\x1b[38;2;139;110;135m"},
		{"loopingliqude", "\x1b[38;2;137;186;253m", "\x1b[38;2;208;248;254m", "\x1b[38;2;141;115;104m"},
		{"heart", "\x1b[38;2;223;93;86m", "\x1b[38;2;255;163;158m", "\x1b[38;2;159;89;97m"},
		{"pinkcat", "\x1b[38;2;254;141;174m", "\x1b[38;2;255;202;214m", "\x1b[38;2;153;81;67m"},
		{"sleepyguy", "\x1b[38;2;208;163;163m", "\x1b[38;2;255;247;234m", "\x1b[38;2;114;86;86m"},
		{"slotmaschine", "\x1b[38;2;240;77;91m", "\x1b[38;2;255;211;105m", "\x1b[38;2;164;150;169m"},
		{"spiderboy", "\x1b[38;2;175;56;70m", "\x1b[38;2;219;73;87m", "\x1b[38;2;108;106;122m"},
	} {
		palette, err := ByName(test.name)
		if err != nil {
			t.Fatal(err)
		}
		if palette.Secondary != test.secondary || palette.Accent != test.accent || palette.Muted != test.muted || palette.Reset != "\x1b[0m" {
			t.Fatalf("%s changed header colors: %+v", test.name, palette)
		}
	}
}
