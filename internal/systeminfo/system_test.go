package systeminfo

import "testing"

func TestSystemInfoProvidesDashboardDetails(t *testing.T) {
	system := SystemInfo()
	if system.Hostname == "" || system.Platform == "" || system.Architecture == "" {
		t.Fatalf("SystemInfo() = %+v, want host and platform details", system)
	}
	if system.Shell == "" || system.Terminal == "" {
		t.Fatalf("SystemInfo() = %+v, want shell and terminal details", system)
	}
}
