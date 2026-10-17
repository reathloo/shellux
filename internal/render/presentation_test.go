package render

import (
	"strings"
	"testing"
	"time"

	"github.com/reathloo/shellux/internal/config"
	"github.com/reathloo/shellux/internal/systeminfo"
	"github.com/reathloo/shellux/internal/theme"
)

func TestPresentationHeaderDoesNotRenderPrivateSnapshotValues(t *testing.T) {
	c := config.Default()
	if err := c.SelectProfile("presentation"); err != nil {
		t.Fatal(err)
	}
	c = c.EffectiveProfile(t.TempDir(), 140, 40)
	snapshot := systeminfo.Snapshot{
		System:      systeminfo.System{Hostname: "PRIVATE-HOST", Shell: "PRIVATE-SHELL", Terminal: "PRIVATE-TERM"},
		Directory:   "/PRIVATE-PATH/project",
		Network:     systeminfo.Network{Connected: true, IPAddress: "PRIVATE-IP", Interface: "PRIVATE-INTERFACE", Name: "PRIVATE-SSID", Connection: "PRIVATE-CONNECTION"},
		LoggedUsers: systeminfo.LoggedUsers{Available: true, Names: []string{"PRIVATE-USER"}},
		Playback:    systeminfo.Playback{Available: true, Title: "PRIVATE-TRACK", Artist: "PRIVATE-ARTIST"},
		Coding:      systeminfo.Coding{Branch: "PRIVATE-BRANCH", Repository: "PRIVATE-REPO", Docker: "PRIVATE-DOCKER", Kubernetes: "PRIVATE-K8S"},
		Timezone:    "PRIVATE-TIMEZONE",
		Now:         time.Date(2026, 10, 17, 12, 34, 0, 0, time.UTC),
	}
	output := NewHeaderLayoutWithConfig(snapshot, "animation", theme.Neon(), c).String()
	for _, secret := range []string{"PRIVATE-HOST", "PRIVATE-SHELL", "PRIVATE-TERM", "PRIVATE-PATH", "PRIVATE-IP", "PRIVATE-INTERFACE", "PRIVATE-SSID", "PRIVATE-CONNECTION", "PRIVATE-USER", "PRIVATE-TRACK", "PRIVATE-ARTIST", "PRIVATE-BRANCH", "PRIVATE-REPO", "PRIVATE-DOCKER", "PRIVATE-K8S", "PRIVATE-TIMEZONE"} {
		if strings.Contains(output, secret) {
			t.Fatalf("Presentation leaked %s", secret)
		}
	}
	for _, value := range []string{"12:34", "UPTIME", "CPU", "MEMORY", "Connected"} {
		if !strings.Contains(output, value) {
			t.Fatalf("Presentation omitted %s", value)
		}
	}
}
