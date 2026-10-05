//go:build !darwin && !linux

package systeminfo

func readVolumeInfo() (Volume, error) {
	return volumeInfoFromDF("/")
}
