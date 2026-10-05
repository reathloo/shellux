//go:build linux

package systeminfo

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func readTemperature() (Temperature, error) {
	paths, err := filepath.Glob("/sys/class/thermal/thermal_zone*/temp")
	if err != nil {
		return Temperature{}, fmt.Errorf("find thermal sensors: %w", err)
	}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		millidegrees, err := strconv.ParseFloat(strings.TrimSpace(string(data)), 64)
		if err != nil {
			continue
		}
		return Temperature{
			Celsius:   millidegrees / 1000,
			Available: true,
			Source:    filepath.Base(filepath.Dir(path)),
		}, nil
	}

	return Temperature{}, nil
}
