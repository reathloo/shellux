package systeminfo

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

const playbackSeparator = "\x1f"

// Playback describes an optional media playback snapshot.
type Playback struct {
	Title     string
	Artist    string
	Position  time.Duration
	Duration  time.Duration
	Playing   bool
	Available bool
	Source    string
}

// PlaybackInfo returns the currently playing local media when available.
func PlaybackInfo() (Playback, error) {
	return readPlayback()
}

func parseSpotifyPlayback(output string) (Playback, error) {
	output = strings.TrimSpace(output)
	if output == "" {
		return Playback{}, nil
	}
	fields := strings.Split(output, playbackSeparator)
	if len(fields) != 5 {
		return Playback{}, fmt.Errorf("parse Spotify playback: expected 5 fields, got %d", len(fields))
	}

	playback := Playback{
		Title:     strings.TrimSpace(fields[0]),
		Artist:    strings.TrimSpace(fields[1]),
		Playing:   strings.EqualFold(strings.TrimSpace(fields[4]), "playing"),
		Available: true,
		Source:    "Spotify",
	}
	if playback.Title == "" || playback.Artist == "" {
		return playback, nil
	}

	durationMilliseconds, err := parseSpotifySeconds(fields[2])
	if err != nil || durationMilliseconds <= 0 {
		return Playback{}, fmt.Errorf("parse Spotify duration %q", fields[2])
	}
	positionSeconds, err := parseSpotifySeconds(fields[3])
	if err != nil || positionSeconds < 0 {
		return Playback{}, fmt.Errorf("parse Spotify position %q", fields[3])
	}

	playback.Position = time.Duration(positionSeconds * float64(time.Second))
	playback.Duration = time.Duration(durationMilliseconds * float64(time.Millisecond))
	if playback.Position > playback.Duration {
		playback.Position = playback.Duration
	}
	return playback, nil
}

func parseSpotifySeconds(value string) (float64, error) {
	return strconv.ParseFloat(strings.ReplaceAll(strings.TrimSpace(value), ",", "."), 64)
}
