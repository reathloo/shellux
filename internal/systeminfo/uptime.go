package systeminfo

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Uptime describes how long the operating system has been running.
type Uptime struct {
	Duration  time.Duration
	Available bool
	Source    string
}

// UptimeInfo returns the current system uptime when the platform exposes it.
func UptimeInfo() (Uptime, error) {
	return readUptime()
}

func parseUptimeSeconds(value string) (time.Duration, error) {
	fields := strings.Fields(value)
	if len(fields) == 0 {
		return 0, fmt.Errorf("parse uptime: missing seconds")
	}
	seconds, err := strconv.ParseFloat(strings.ReplaceAll(fields[0], ",", "."), 64)
	if err != nil || seconds < 0 {
		return 0, fmt.Errorf("parse uptime seconds %q", fields[0])
	}
	return time.Duration(seconds * float64(time.Second)), nil
}
