package systeminfo

import (
	"testing"
	"time"
)

func TestParseUptimeSeconds(t *testing.T) {
	duration, err := parseUptimeSeconds("93784,5 123.4\n")
	if err != nil {
		t.Fatalf("parseUptimeSeconds() error = %v", err)
	}
	if got, want := duration, 26*time.Hour+3*time.Minute+4*time.Second+500*time.Millisecond; got != want {
		t.Fatalf("duration = %v, want %v", got, want)
	}
}

func TestParseUptimeSecondsRejectsInvalidValues(t *testing.T) {
	if _, err := parseUptimeSeconds("-1"); err == nil {
		t.Fatal("parseUptimeSeconds() accepted negative uptime")
	}
}
