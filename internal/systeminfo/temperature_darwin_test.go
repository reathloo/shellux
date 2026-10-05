//go:build darwin

package systeminfo

import "testing"

func TestParseThermalState(t *testing.T) {
	output := "**** Thermal pressure ****\nCurrent pressure level: Nominal\n"
	if got, want := parseThermalState(output), "Nominal"; got != want {
		t.Fatalf("parseThermalState() = %q, want %q", got, want)
	}
}

func TestParseThermalStateSupportsLegacyLabel(t *testing.T) {
	if got, want := parseThermalState("CPU Thermal pressure: Serious\n"), "Serious"; got != want {
		t.Fatalf("parseThermalState() = %q, want %q", got, want)
	}
}

func TestParseThermalStateUnavailable(t *testing.T) {
	if got := parseThermalState("no sensor data"); got != "" {
		t.Fatalf("parseThermalState() = %q, want empty", got)
	}
}
