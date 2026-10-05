//go:build linux

package systeminfo

import (
	"fmt"
	"os"
)

func readLoadAverage() (float64, error) {
	data, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return 0, fmt.Errorf("read /proc/loadavg: %w", err)
	}

	return parseLoadAverage(string(data))
}
