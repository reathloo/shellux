//go:build darwin

package systeminfo

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

const spotifyPlaybackScript = `if application id "com.spotify.client" is running then
	tell application id "com.spotify.client"
		set state to player state as text
		try
			set albumName to ""
			try
				set albumName to album of current track
			end try
			return (name of current track) & (ASCII character 31) & (artist of current track) & (ASCII character 31) & albumName & (ASCII character 31) & (duration of current track) & (ASCII character 31) & (player position) & (ASCII character 31) & state
		on error
			return (ASCII character 31) & (ASCII character 31) & (ASCII character 31) & "0" & (ASCII character 31) & "0" & (ASCII character 31) & state
		end try
	end tell
end if`

func readPlayback() (Playback, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 750*time.Millisecond)
	defer cancel()

	output, err := exec.CommandContext(ctx, "osascript", "-e", spotifyPlaybackScript).Output()
	if err != nil {
		return Playback{}, nil
	}
	return parseSpotifyPlayback(string(output))
}

const spotifyControlPrefix = `if application id "com.spotify.client" is not running then return "NOT_RUNNING"
tell application id "com.spotify.client"
`

const spotifyControlSuffix = `
end tell
return "OK"`

func spotifyControlScript(action PlaybackAction) string {
	var command string
	switch action {
	case PlaybackPrevious:
		command = "previous track"
	case PlaybackNext:
		command = "next track"
	case PlaybackToggle:
		command = "playpause"
	case PlaybackRewind10:
		command = `set newPosition to (player position) - 10
if newPosition < 0 then set newPosition to 0
set player position to newPosition`
	case PlaybackForward10:
		command = `set newPosition to (player position) + 10
try
	set trackSeconds to (duration of current track) / 1000
	if newPosition > trackSeconds then set newPosition to trackSeconds
end try
set player position to newPosition`
	}
	return spotifyControlPrefix + command + spotifyControlSuffix
}

func controlPlayback(action PlaybackAction) error {
	return runSpotifyControlScript(spotifyControlScript(action))
}

func spotifySeekScript(position time.Duration) string {
	seconds := strconv.FormatFloat(position.Seconds(), 'f', 3, 64)
	return spotifyControlPrefix + `set newPosition to ` + seconds + `
try
	set trackSeconds to (duration of current track) / 1000
	if newPosition > trackSeconds then set newPosition to trackSeconds
end try
set player position to newPosition` + spotifyControlSuffix
}

func seekPlayback(position time.Duration) error {
	return runSpotifyControlScript(spotifySeekScript(position))
}

func runSpotifyControlScript(script string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	output, err := exec.CommandContext(ctx, "osascript", "-e", script).CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if len(message) > 240 {
			message = message[:240]
		}
		if message != "" {
			return fmt.Errorf("control Spotify: %s", message)
		}
		return fmt.Errorf("control Spotify: %w", err)
	}
	if strings.TrimSpace(string(output)) == "NOT_RUNNING" {
		return ErrPlaybackUnavailable
	}
	return nil
}
