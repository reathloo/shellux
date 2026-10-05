//go:build darwin && !cgo

package systeminfo

import "fmt"

func nativeVolumeInfo() (Volume, error) {
	return Volume{}, fmt.Errorf("native macOS volume capacity requires cgo")
}
