package systeminfo

import (
	"fmt"
	"runtime"
	"strconv"
	"strings"
)

// CPU describes the CPU values currently available to Shellux.
type CPU struct {
	Cores          int
	LoadAverage    float64
	UsagePercent   float64
	UsageAvailable bool
	TopProcess     Process
}

// CPUInfo returns a snapshot of the host CPU information.
func CPUInfo() (CPU, error) {
	loadAverage, err := readLoadAverage()
	if err != nil {
		return CPU{}, err
	}

	usage, available := cpuUsage()
	return CPU{
		Cores:          runtime.NumCPU(),
		LoadAverage:    loadAverage,
		UsagePercent:   usage,
		UsageAvailable: available,
	}, nil
}

func parseLoadAverage(value string) (float64, error) {
	fields := strings.Fields(value)
	if len(fields) == 0 {
		return 0, fmt.Errorf("load average is empty")
	}

	for _, field := range fields {
		firstField := strings.Trim(field, "{}")
		if firstField == "" {
			continue
		}

		// macOS localizes sysctl output and uses a decimal comma for locales
		// such as de_DE. Normalize it before parsing; Linux already uses a dot.
		normalizedField := strings.ReplaceAll(firstField, ",", ".")
		loadAverage, err := strconv.ParseFloat(normalizedField, 64)
		if err != nil {
			continue
		}
		if loadAverage < 0 {
			return 0, fmt.Errorf("load average cannot be negative: %v", loadAverage)
		}

		return loadAverage, nil
	}

	return 0, fmt.Errorf("parse load average from %q", value)
}
