//go:build darwin && !cgo

package systeminfo

import "fmt"

func readDiskCounters() (diskCounters, error) {
	return diskCounters{}, fmt.Errorf("disk I/O counters require cgo on macOS")
}
