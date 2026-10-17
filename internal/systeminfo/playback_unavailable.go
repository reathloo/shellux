//go:build !darwin

package systeminfo

import "time"

func readPlayback() (Playback, error) {
	return Playback{}, nil
}

func controlPlayback(PlaybackAction) error {
	return ErrPlaybackUnavailable
}

func seekPlayback(time.Duration) error {
	return ErrPlaybackUnavailable
}
