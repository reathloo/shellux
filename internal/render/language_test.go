package render

import (
	"github.com/reathloo/shellux/internal/locale"
	"github.com/reathloo/shellux/internal/systeminfo"
	"github.com/reathloo/shellux/internal/theme"
	"strings"
	"testing"
	"time"
)

func TestLocalizedHeaderPreservesUserContent(t *testing.T) {
	snapshot := systeminfo.Snapshot{Now: time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC), Network: systeminfo.Network{Connected: true, Name: "Connected", Interface: "en0"}, Directory: "/tmp/Connected", Memory: systeminfo.Memory{Total: 100, Available: 50, Pressure: "Warning"}, Playback: systeminfo.Playback{Available: true, Artist: "Warning", Title: "Connected"}}
	visible := map[string]bool{"date": true, "network-status.name": true, "ram.pressure": true, "directory": true}
	for _, tc := range []struct {
		lang  locale.Language
		words []string
	}{
		{locale.English, []string{"NETWORK", "Connected", "2026-10-05", "Warning"}},
		{locale.German, []string{"NETZWERK", "Verbunden", "05.10.2026", "Warnung"}},
	} {
		text := stripANSI(NewHeaderLayout(snapshot, "", theme.Theme{}, visible, tc.lang).String())
		for _, word := range append(tc.words, "Warning — Connected", "/tmp/Connected") {
			if !strings.Contains(text, word) {
				t.Fatalf("%s missing %q: %s", tc.lang, word, text)
			}
		}
	}
	if snapshot.Memory.Pressure != "Warning" {
		t.Fatal("renderer mutated snapshot")
	}
	if got := networkStatusValueParts(systeminfo.Network{}, true, false, false, false, false, locale.German); got != "Nicht verbunden" {
		t.Fatal(got)
	}
}
