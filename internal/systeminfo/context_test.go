package systeminfo

import (
	"path/filepath"
	"testing"
	"time"
)

func TestDirectoryInfo(t *testing.T) {
	directory, err := DirectoryInfo()
	if err != nil {
		t.Fatalf("DirectoryInfo() error = %v", err)
	}
	if directory == "" || !filepath.IsAbs(directory) {
		t.Fatalf("DirectoryInfo() = %q, want an absolute path", directory)
	}
}

func TestTimeInfo(t *testing.T) {
	before := time.Now()
	got := TimeInfo()
	after := time.Now()
	if got.Before(before) || got.After(after) {
		t.Fatalf("TimeInfo() = %v, want current local time", got)
	}
}

func TestTimezoneNameUsesConfiguredZone(t *testing.T) {
	for _, tc := range []struct{ zone, want string }{
		{"Europe/Berlin", "Berlin"}, {"America/Argentina/Buenos_Aires", "Buenos Aires"},
		{"America/Los_Angeles", "Los Angeles"}, {"UTC", "UTC"}, {"CET", "CET"},
	} {
		now := time.Date(2026, 9, 27, 12, 0, 0, 0, time.FixedZone(tc.zone, 3600))
		if got := TimezoneName(now); got != tc.want {
			t.Fatalf("%s = %q, want %q", tc.zone, got, tc.want)
		}
	}
}
