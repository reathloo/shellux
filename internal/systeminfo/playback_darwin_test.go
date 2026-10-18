//go:build darwin

package systeminfo

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSpotifyControlScriptsCompile(t *testing.T) {
	// AppleScript resolves Spotify's scripting dictionary at compile time.
	// CI macOS runners without Spotify cannot validate these scripts.
	if output, err := exec.Command("osascript", "-e", `application id "com.spotify.client" is running`).CombinedOutput(); err != nil {
		t.Skipf("Spotify scripting target unavailable: %v: %s", err, output)
	}
	readScript := filepath.Join(t.TempDir(), "read.scpt")
	if output, err := exec.Command("osacompile", "-o", readScript, "-e", spotifyPlaybackScript).CombinedOutput(); err != nil {
		t.Fatalf("compile playback AppleScript: %v: %s", err, output)
	}
	for _, action := range []PlaybackAction{PlaybackPrevious, PlaybackRewind10, PlaybackForward10, PlaybackNext, PlaybackToggle} {
		t.Run(string(action), func(t *testing.T) {
			script := spotifyControlScript(action)
			if !strings.Contains(script, `is not running then return "NOT_RUNNING"`) {
				t.Fatal("control script may start Spotify")
			}
			compiled := filepath.Join(t.TempDir(), "control.scpt")
			if output, err := exec.Command("osacompile", "-o", compiled, "-e", script).CombinedOutput(); err != nil {
				t.Fatalf("compile AppleScript: %v: %s", err, output)
			}
		})
	}
	compiled := filepath.Join(t.TempDir(), "seek.scpt")
	if output, err := exec.Command("osacompile", "-o", compiled, "-e", spotifySeekScript(90*time.Second)).CombinedOutput(); err != nil {
		t.Fatalf("compile seek AppleScript: %v: %s", err, output)
	}
}
