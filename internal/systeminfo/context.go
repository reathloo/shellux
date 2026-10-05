package systeminfo

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

// DirectoryInfo returns the current working directory.
func DirectoryInfo() (string, error) {
	return os.Getwd()
}

// TimeInfo returns the current local time.
func TimeInfo() time.Time {
	return time.Now()
}

// TimezoneName shows the configured zone's city when its IANA name is known.
// Go's default location is often just "Local", so use the system zoneinfo link
// on macOS/Linux. Never infer a city from an ambiguous abbreviation like CET.
func TimezoneName(now time.Time) string {
	name := now.Location().String()
	if name != "" && name != "Local" && !filepath.IsAbs(name) {
		return timezoneLabel(name)
	}
	path := "/etc/localtime"
	if filepath.IsAbs(name) {
		path = name
	}
	if target, err := filepath.EvalSymlinks(path); err == nil {
		if _, zone, ok := strings.Cut(filepath.ToSlash(target), "/zoneinfo/"); ok {
			return timezoneLabel(zone)
		}
	}
	if data, err := os.ReadFile("/etc/timezone"); err == nil {
		zone := strings.TrimSpace(string(data))
		if zone != "" {
			return timezoneLabel(zone)
		}
	}
	zone, _ := now.Zone()
	return zone
}

func timezoneLabel(name string) string {
	return strings.ReplaceAll(filepath.Base(name), "_", " ")
}
