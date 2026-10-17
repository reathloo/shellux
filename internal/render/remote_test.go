package render

import (
	"strings"
	"testing"
	"time"

	"github.com/reathloo/shellux/internal/config"
	"github.com/reathloo/shellux/internal/systeminfo"
	"github.com/reathloo/shellux/internal/theme"
)

func TestRemoteHeaderShowsHostAndConnectionContext(t *testing.T) {
	c := config.Default()
	if err := c.SelectProfile("remote"); err != nil {
		t.Fatal(err)
	}
	c = c.EffectiveProfile(t.TempDir(), 140, 40)
	snapshot := systeminfo.Snapshot{
		System:    systeminfo.System{Hostname: "remote-host", Shell: "zsh", Terminal: "xterm"},
		Directory: "/srv/project",
		Network:   systeminfo.Network{Connected: true, Interface: "eth0", IPAddress: "192.0.2.10"},
		Remote:    systeminfo.RemoteSession{User: "deploy", Client: "192.0.2.1:54321", Server: "192.0.2.10:22", TTY: "/dev/pts/3", Term: "xterm-256color"},
		Playback:  systeminfo.Playback{Available: true, Title: "hidden-song"},
		Now:       time.Date(2026, 10, 17, 12, 34, 0, 0, time.UTC),
	}
	output := NewHeaderLayoutWithConfig(snapshot, "animation", theme.Neon(), c).String()
	for _, value := range []string{"deploy@remote-host", "192.0.2.1:54321", "192.0.2.10:22", "/dev/pts/3", "xterm-256color", "/srv/project", "eth0", "Connected", "CPU", "MEMORY"} {
		if !strings.Contains(output, value) {
			t.Fatalf("Remote omitted %s", value)
		}
	}
	if strings.Contains(output, "hidden-song") {
		t.Fatal("Remote showed unrelated media")
	}
	if strings.Index(output, "REMOTE") > strings.Index(output, "RESOURCES") || strings.Index(output, "RESOURCES") > strings.Index(output, "SESSION") {
		t.Fatal("Remote group order is wrong")
	}
}
