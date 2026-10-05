//go:build !darwin

package systeminfo

func readPlayback() (Playback, error) {
	return Playback{}, nil
}
