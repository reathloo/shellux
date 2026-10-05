//go:build darwin

package systeminfo

import (
	"context"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"
)

const thermalRefreshInterval = 10 * time.Second

var thermalStatePattern = regexp.MustCompile(`(?im)(?:thermal pressure|current pressure level):\s*([[:alpha:]]+)`)

var thermalCache struct {
	sync.Mutex
	checkedAt time.Time
	value     Temperature
}

func readTemperature() (Temperature, error) {
	thermalCache.Lock()
	defer thermalCache.Unlock()

	if time.Since(thermalCache.checkedAt) < thermalRefreshInterval {
		return thermalCache.value, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	output, err := exec.CommandContext(ctx, "/usr/bin/sudo", "-n", "/usr/bin/powermetrics", "-n", "1", "-i", "100", "--samplers", "thermal").Output()
	if err != nil {
		thermalCache.checkedAt = time.Now()
		thermalCache.value = Temperature{}
		return thermalCache.value, nil
	}

	state := parseThermalState(string(output))
	thermalCache.checkedAt = time.Now()
	thermalCache.value = Temperature{
		State:     state,
		Available: state != "",
		Source:    "powermetrics",
	}
	return thermalCache.value, nil
}

func parseThermalState(output string) string {
	match := thermalStatePattern.FindStringSubmatch(output)
	if len(match) != 2 {
		return ""
	}
	return strings.ToUpper(match[1][:1]) + strings.ToLower(match[1][1:])
}
