package systeminfo

import (
	"os"
	"path/filepath"
	"runtime"
)

// System describes stable host and terminal details for the dashboard.
type System struct {
	Hostname     string
	Platform     string
	Architecture string
	Shell        string
	Terminal     string
}

// SystemInfo returns best-effort static information about the current session.
func SystemInfo() System {
	hostname, err := os.Hostname()
	if err != nil || hostname == "" {
		hostname = "unknown host"
	}

	platform := runtime.GOOS
	if platform == "darwin" {
		platform = "macOS"
	}

	shell := filepath.Base(os.Getenv("SHELL"))
	if shell == "." || shell == "" {
		shell = "unknown"
	}
	terminal := os.Getenv("TERM_PROGRAM")
	if terminal == "" {
		terminal = os.Getenv("TERM")
	}
	if terminal == "" {
		terminal = "terminal"
	}

	return System{
		Hostname:     hostname,
		Platform:     platform,
		Architecture: runtime.GOARCH,
		Shell:        shell,
		Terminal:     terminal,
	}
}
