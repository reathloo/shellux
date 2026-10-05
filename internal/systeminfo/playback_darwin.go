//go:build darwin

package systeminfo

import (
	"context"
	"os/exec"
	"time"
)

const spotifyPlaybackScript = `if application id "com.spotify.client" is running then
	tell application id "com.spotify.client"
		set state to player state as text
		try
			return (name of current track) & (ASCII character 31) & (artist of current track) & (ASCII character 31) & (duration of current track) & (ASCII character 31) & (player position) & (ASCII character 31) & state
		on error
			return (ASCII character 31) & (ASCII character 31) & "0" & (ASCII character 31) & "0" & (ASCII character 31) & state
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
