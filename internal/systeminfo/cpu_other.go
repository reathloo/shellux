//go:build !darwin && !linux

package systeminfo

import "fmt"

func readLoadAverage() (float64, error) {
	return 0, fmt.Errorf("load average is not supported on this operating system")
}
