//go:build darwin

package systeminfo

import "testing"

func TestParseDefaultRouteInterface(t *testing.T) {
	output := "   route to: default\ndestination: default\n    gateway: 192.168.0.1\n  interface: en0\n"
	if got, want := parseDefaultRouteInterface(output), "en0"; got != want {
		t.Fatalf("parseDefaultRouteInterface() = %q, want %q", got, want)
	}
}
