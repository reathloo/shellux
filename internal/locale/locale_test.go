package locale

import "testing"

func TestLocalePrecedenceAndFallback(t *testing.T) {
	for _, tc := range []struct {
		all, messages, lang, system string
		want                        Language
	}{
		{"de_DE.UTF-8", "en_US", "en_US", "en", German},
		{"", "de_AT", "en_US", "en", German},
		{"", "", "de-CH", "en", German},
		{"C", "de_DE", "de_DE", "de", English},
		{"", "", "fr_FR.UTF-8", "de", English},
		{"", "", "", "de-DE", German},
		{"", "", "", "", English},
	} {
		env := map[string]string{"LC_ALL": tc.all, "LC_MESSAGES": tc.messages, "LANG": tc.lang}
		got := detect(func(k string) string { return env[k] }, func() string { return tc.system })
		if got != tc.want {
			t.Fatalf("%+v: got %s", tc, got)
		}
	}
}
func TestExplicitChoiceAndUnknownContent(t *testing.T) {
	t.Setenv("LC_ALL", "de_DE.UTF-8")
	if Resolve("en") != English || Resolve("auto") != German {
		t.Fatal("override or auto ignored")
	}
	t.Setenv("LC_ALL", "C")
	if Resolve("de") != German {
		t.Fatal("override ignored")
	}
	if German.Text("Connected") != "Verbunden" || German.Text("my-custom-theme") != "my-custom-theme" {
		t.Fatal("incorrect translation boundary")
	}
}
