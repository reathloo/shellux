package systeminfo

import (
	"testing"
	"time"
)

func TestParseSpotifyPlayback(t *testing.T) {
	playback, err := parseSpotifyPlayback("Get Lucky\x1fDaft Punk\x1f369500\x1f84,25\x1fplaying")
	if err != nil {
		t.Fatalf("parseSpotifyPlayback() error = %v", err)
	}
	if !playback.Available || !playback.Playing {
		t.Fatalf("parseSpotifyPlayback() availability = %+v, want active playback", playback)
	}
	if got, want := playback.Position, 84*time.Second+250*time.Millisecond; got != want {
		t.Fatalf("position = %v, want %v", got, want)
	}
	if got, want := playback.Duration, 369*time.Second+500*time.Millisecond; got != want {
		t.Fatalf("duration = %v, want %v", got, want)
	}
}

func TestParseSpotifyPlaybackHandlesUnavailableAndInvalidValues(t *testing.T) {
	if playback, err := parseSpotifyPlayback(""); err != nil || playback.Available {
		t.Fatalf("empty playback = %+v, %v; want unavailable, nil", playback, err)
	}
	if _, err := parseSpotifyPlayback("song\x1fartist\x1fnope\x1f1\x1fplaying"); err == nil {
		t.Fatal("parseSpotifyPlayback() accepted invalid duration")
	}
	playback, err := parseSpotifyPlayback("\x1f\x1f0\x1f0\x1fpaused")
	if err != nil || !playback.Available || playback.Playing || playback.Title != "" {
		t.Fatalf("inactive playback = %+v, %v; want available paused state", playback, err)
	}
}
