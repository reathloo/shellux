//go:build darwin

package systeminfo

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"time"
)

var darwinBootTimePattern = regexp.MustCompile(`sec = ([0-9]+)`)

func readUptime() (Uptime, error) {
	output, err := exec.Command("sysctl", "-n", "kern.boottime").Output()
	if err != nil {
		return Uptime{}, nil
	}
	bootTime, err := parseDarwinBootTime(string(output))
	if err != nil {
		return Uptime{}, err
	}
	duration := time.Since(bootTime)
	if duration < 0 {
		return Uptime{}, fmt.Errorf("boot time is in the future")
	}
	return Uptime{Duration: duration, Available: true, Source: "sysctl"}, nil
}

func parseDarwinBootTime(value string) (time.Time, error) {
	match := darwinBootTimePattern.FindStringSubmatch(value)
	if len(match) != 2 {
		return time.Time{}, fmt.Errorf("parse kern.boottime")
	}
	seconds, err := strconv.ParseInt(match[1], 10, 64)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse boot time: %w", err)
	}
	return time.Unix(seconds, 0), nil
}
