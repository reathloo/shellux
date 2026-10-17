package render

import (
	"strings"
	"testing"
	"time"

	"github.com/reathloo/shellux/internal/config"
	"github.com/reathloo/shellux/internal/systeminfo"
	"github.com/reathloo/shellux/internal/theme"
)

func TestMinimalLayoutKeepsMetricsVisibleInNarrowTerminal(t *testing.T) {
	c := config.Default().EffectiveProfile(t.TempDir(), 40, 12)
	if c.RuntimeProfile != "" {
		// The default manual selection has no active preset.
		t.Fatal("manual standard unexpectedly activated Minimal")
	}
	if err := c.SelectProfile("minimal"); err != nil {
		t.Fatal(err)
	}
	c = c.EffectiveProfile(t.TempDir(), 40, 12)
	snapshot := systeminfo.Snapshot{Directory: "/tmp/project", Now: time.Date(2026, 10, 17, 12, 34, 0, 0, time.UTC)}
	layout := NewHeaderLayoutWithConfig(snapshot, "•", theme.Neon(), c).FitTerminal(40, 12)
	output := layout.String()
	for _, want := range []string{"Shellux", "12:34", "PATH", "project", "CPU", "MEMORY"} {
		if !strings.Contains(output, want) {
			t.Fatalf("minimal layout missing %q: %s", want, output)
		}
	}
	if strings.Contains(output, "╭─") || layout.EyeWidth != 0 || layout.DashboardWidth > 40 || layout.Height() > 10 {
		t.Fatalf("minimal layout still uses the large dashboard: %+v", layout)
	}
}
