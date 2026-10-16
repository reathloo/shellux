//go:build !darwin && !linux

package systeminfo

import "fmt"

func readDiskCounters() (diskCounters, error) {
	return diskCounters{}, fmt.Errorf("disk I/O counters unavailable")
}
