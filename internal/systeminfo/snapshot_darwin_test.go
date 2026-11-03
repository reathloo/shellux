//go:build darwin

package systeminfo

import (
	"testing"

	"github.com/reathloo/shellux/internal/config"
)

func TestPlaybackQueryFollowsVisibleEntries(t *testing.T) {
	for _, tc := range []struct{ spotify, progress bool }{{false, false}, {true, false}, {false, true}, {true, true}} {
		c := config.Default()
		c.Visible["spotify"], c.Visible["progress"] = tc.spotify, tc.progress
		c.Visible["temperature"], c.Visible["battery"], c.Visible["volume"] = false, false, false
		calls := 0
		playbackInfo := func() (Playback, error) {
			calls++
			return Playback{Available: true, Source: "Spotify"}, nil
		}
		snapshot, err := snapshotInfoWithDisplay(c.Visible, playbackInfo)
		if err != nil {
			t.Fatal(err)
		}
		wantCalls := 0
		if tc.spotify || tc.progress {
			wantCalls = 1
		}
		if calls != wantCalls || snapshot.Playback.Available != (wantCalls != 0) {
			t.Fatalf("visibility %+v: calls=%d playback=%+v", tc, calls, snapshot.Playback)
		}
		if snapshot.Volume != (Volume{}) || snapshot.Battery != (Battery{}) || snapshot.Temperature != (Temperature{}) {
			t.Fatal("hidden optional metrics were collected")
		}
	}
}
