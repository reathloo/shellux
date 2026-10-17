package render

import (
	"strings"
	"testing"
	"time"

	"github.com/reathloo/shellux/internal/config"
	"github.com/reathloo/shellux/internal/locale"
	"github.com/reathloo/shellux/internal/systeminfo"
	"github.com/reathloo/shellux/internal/theme"
)

func TestCodingDashboardUsesEnabledParts(t *testing.T) {
	c := config.Default()
	for name := range c.Visible {
		c.Visible[name] = false
	}
	for _, name := range []string{"git-branch", "git-sync", "runtimes", "dev-context", "last-command"} {
		c.Visible[name] = true
	}
	snapshot := systeminfo.Snapshot{Now: time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC), Coding: systeminfo.Coding{
		Branch: "main", Repository: "shellux", Ahead: 2, Behind: 1, SyncAvailable: true,
		Go: "1.25", Node: "22", Python: "3.13", Java: "21",
		Docker: "default", Kubernetes: "dev", CommandSeconds: 0, CommandExit: 7, CommandAvailable: true,
	}}
	output := NewHeaderLayoutWithConfig(snapshot, "eye", theme.DarkLuxury(), c, locale.English).String()
	for _, want := range []string{"CODING", "main", "repo shellux", "↑2", "↓1", "Go 1.25", "Node 22", "Python 3.13", "Java 21", "Docker default", "K8s dev", "<1s", "exit 7"} {
		if !strings.Contains(output, want) {
			t.Fatalf("coding header missing %q: %s", want, output)
		}
	}
	c.Visible["git-branch.repository"] = false
	c.Visible["runtimes.java"] = false
	output = NewHeaderLayoutWithConfig(snapshot, "eye", theme.DarkLuxury(), c, locale.English).String()
	if strings.Contains(output, "repo shellux") || strings.Contains(output, "Java 21") {
		t.Fatal("disabled coding details remained visible")
	}
}
