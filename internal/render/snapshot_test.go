package render

import (
	"strings"
	"testing"
	"time"

	"github.com/reathloo/shellux/internal/config"
	"github.com/reathloo/shellux/internal/systeminfo"
	"github.com/reathloo/shellux/internal/theme"
	"github.com/reathloo/shellux/internal/version"
)

func TestSnapshotRendersCaptureAndOmitsPersonalValues(t *testing.T) {
	c := config.Default()
	if err := c.SelectProfile("snapshot"); err != nil {
		t.Fatal(err)
	}
	c = c.EffectiveProfile(t.TempDir(), 140, 40)
	now := time.Date(2026, 10, 17, 14, 23, 7, 0, time.FixedZone("CEST", 2*60*60))
	snapshot := systeminfo.Snapshot{
		Now: now, Directory: "/PRIVATE-PATH", System: systeminfo.System{
			Platform: "macOS", Architecture: "arm64", Hostname: "PRIVATE-HOST", Shell: "zsh", Terminal: "xterm-256color",
		}, Network: systeminfo.Network{Connected: true, IPAddress: "PRIVATE-IP", Name: "PRIVATE-SSID"},
	}
	output := stripANSI(NewHeaderLayoutWithConfig(snapshot, "eye", theme.Neon(), c).String())
	for _, value := range []string{"SNAPSHOT", "2026-10-17 14:23:07 +02:00", version.String(), "macOS", "arm64"} {
		if !strings.Contains(output, value) {
			t.Fatalf("Snapshot is missing %q: %s", value, output)
		}
	}
	for _, value := range []string{"PRIVATE-PATH", "PRIVATE-HOST", "PRIVATE-IP", "PRIVATE-SSID"} {
		if strings.Contains(output, value) {
			t.Fatalf("Snapshot exposed %q", value)
		}
	}
}
