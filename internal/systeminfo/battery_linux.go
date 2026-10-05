//go:build linux

package systeminfo

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func readBattery() (Battery, error) {
	paths, err := filepath.Glob("/sys/class/power_supply/BAT*/capacity")
	if err != nil {
		return Battery{}, fmt.Errorf("find batteries: %w", err)
	}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		percentage, err := strconv.Atoi(strings.TrimSpace(string(data)))
		if err != nil || percentage < 0 || percentage > 100 {
			continue
		}

		statusData, _ := os.ReadFile(filepath.Join(filepath.Dir(path), "status"))
		status := strings.TrimSpace(string(statusData))
		remaining := time.Duration(0)
		if status == "Discharging" {
			remaining = linuxBatteryRemaining(filepath.Dir(path))
		}
		return Battery{
			Percentage:         percentage,
			Charging:           status == "Charging" || status == "Full",
			Available:          true,
			Source:             filepath.Base(filepath.Dir(path)),
			Remaining:          remaining,
			RemainingAvailable: remaining > 0,
		}, nil
	}

	return Battery{}, nil
}

func linuxBatteryRemaining(dir string) time.Duration {
	read := func(name string) float64 {
		data, _ := os.ReadFile(filepath.Join(dir, name))
		value, _ := strconv.ParseFloat(strings.TrimSpace(string(data)), 64)
		return value
	}
	if seconds := read("time_to_empty_now"); seconds > 0 && seconds < 3600000 {
		return time.Duration(seconds * float64(time.Second))
	}
	for _, pair := range [][2]string{{"energy_now", "power_now"}, {"charge_now", "current_now"}} {
		energy, rate := read(pair[0]), read(pair[1])
		if energy > 0 && rate > 0 {
			hours := energy / rate
			if hours > 0 && hours < 1000 {
				return time.Duration(hours * float64(time.Hour))
			}
		}
	}
	return 0
}
