//go:build linux

package systeminfo

func readVolumeInfo() (Volume, error) {
	return volumeInfoFromDF("/")
}
