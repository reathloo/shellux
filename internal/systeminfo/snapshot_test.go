package systeminfo

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/reathloo/shellux/internal/config"
)

func TestOptionalCollectorsOnlyRunWhenEnabled(t *testing.T) {
	dir := t.TempDir()
	counter := filepath.Join(dir, "calls")
	for _, name := range []string{"ps", "networksetup", "ping"} {
		script := "#!/bin/sh\nprintf '" + name + "\\n' >> '" + counter + "'\nprintf '13.0 fixture-worker\\n'\n"
		if err := os.WriteFile(filepath.Join(dir, name), []byte(script), 0700); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	key := "ps\x00-axo\x00pcpu=,comm="
	detailCommands.Lock()
	delete(detailCommands.values, key)
	detailCommands.Unlock()
	t.Cleanup(func() { detailCommands.Lock(); delete(detailCommands.values, key); detailCommands.Unlock() })
	c := config.Default()
	for _, switches := range [][2]bool{{true, false}, {false, true}, {false, false}} {
		c.Visible["network-traffic"], c.Visible["network-traffic.latency"] = switches[0], switches[1]
		if _, err := SnapshotInfoWithDisplay(c.Visible); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(counter); !os.IsNotExist(err) {
			t.Fatal("hidden details ran extra commands")
		}
	}
	if err := c.ChangeDisplay("cpu", "top", true); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		snapshot, err := SnapshotInfoWithDisplay(c.Visible)
		if err != nil || snapshot.CPU.TopProcess.Name != "fixture-worker" {
			t.Fatalf("enabled collector: %+v, %v", snapshot.CPU.TopProcess, err)
		}
	}
	data, _ := os.ReadFile(counter)
	if string(data) != "ps\n" {
		t.Fatalf("extra/repeated commands: %q", data)
	}
}

func TestSnapshotInfo(t *testing.T) {
	snapshot, err := SnapshotInfo()
	if err != nil {
		t.Fatalf("SnapshotInfo() error = %v", err)
	}
	if snapshot.Directory == "" || snapshot.Now.IsZero() {
		t.Fatalf("SnapshotInfo() = %+v, want directory and time", snapshot)
	}
}
