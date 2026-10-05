//go:build linux

package systeminfo

import "os"

func readUptime() (Uptime, error) {
	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return Uptime{}, nil
	}
	duration, err := parseUptimeSeconds(string(data))
	if err != nil {
		return Uptime{}, err
	}
	return Uptime{Duration: duration, Available: true, Source: "/proc/uptime"}, nil
}
