package systeminfo

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const playbackSeparator = "\x1f"

type PlaybackAction string

const (
	PlaybackPrevious  PlaybackAction = "previous"
	PlaybackRewind10  PlaybackAction = "rewind-10"
	PlaybackForward10 PlaybackAction = "forward-10"
	PlaybackNext      PlaybackAction = "next"
	PlaybackToggle    PlaybackAction = "toggle"
)

var ErrPlaybackUnavailable = errors.New("Spotify is not running")

// ControlPlayback changes the local Spotify player without starting the app.
func ControlPlayback(action PlaybackAction) error {
	switch action {
	case PlaybackPrevious, PlaybackRewind10, PlaybackForward10, PlaybackNext, PlaybackToggle:
		return controlPlayback(action)
	default:
		return fmt.Errorf("unknown playback action %q", action)
	}
}

// SeekPlayback moves the local Spotify player to an absolute song position.
func SeekPlayback(position time.Duration) error {
	if position < 0 {
		return fmt.Errorf("negative playback position")
	}
	return seekPlayback(position)
}

// Playback describes an optional media playback snapshot.
type Playback struct {
	Title     string
	Artist    string
	Album     string
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
	if len(fields) != 6 {
		return Playback{}, fmt.Errorf("parse Spotify playback: expected 6 fields, got %d", len(fields))
	}

	playback := Playback{
		Title:     strings.TrimSpace(fields[0]),
		Artist:    strings.TrimSpace(fields[1]),
		Album:     strings.TrimSpace(fields[2]),
		Playing:   strings.EqualFold(strings.TrimSpace(fields[5]), "playing"),
		Available: true,
		Source:    "Spotify",
	}
	if playback.Title == "" || playback.Artist == "" {
		return playback, nil
	}

	durationMilliseconds, err := parseSpotifySeconds(fields[3])
	if err != nil || durationMilliseconds <= 0 {
		return Playback{}, fmt.Errorf("parse Spotify duration %q", fields[3])
	}
	positionSeconds, err := parseSpotifySeconds(fields[4])
	if err != nil || positionSeconds < 0 {
		return Playback{}, fmt.Errorf("parse Spotify position %q", fields[4])
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
