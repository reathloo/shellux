//go:build darwin

package systeminfo

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/reathloo/shellux/internal/config"
)

func TestPlaybackQueryFollowsVisibleEntries(t *testing.T) {
	dir := t.TempDir()
	counter := filepath.Join(dir, "calls")
	script := "#!/bin/sh\nprintf x >> '" + counter + "'\nprintf 'Track\\037Artist\\03760000\\0371\\037playing'\n"
	if err := os.WriteFile(filepath.Join(dir, "osascript"), []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	for _, tc := range []struct{ spotify, progress bool }{{false, false}, {true, false}, {false, true}, {true, true}} {
		c := config.Default()
		c.Visible["spotify"], c.Visible["progress"] = tc.spotify, tc.progress
		c.Visible["temperature"], c.Visible["battery"], c.Visible["volume"] = false, false, false
		if err := os.WriteFile(counter, nil, 0600); err != nil {
			t.Fatal(err)
		}
		snapshot, err := SnapshotInfoWithDisplay(c.Visible)
		if err != nil {
			t.Fatal(err)
		}
		calls, err := os.ReadFile(counter)
		if err != nil {
			t.Fatal(err)
		}
		want := ""
		if tc.spotify || tc.progress {
			want = "x"
		}
		if string(calls) != want || snapshot.Playback.Available != (want != "") {
			t.Fatalf("visibility %+v: calls=%q playback=%+v", tc, calls, snapshot.Playback)
		}
		if snapshot.Volume != (Volume{}) || snapshot.Battery != (Battery{}) || snapshot.Temperature != (Temperature{}) {
			t.Fatal("hidden optional metrics were collected")
		}
	}
}
