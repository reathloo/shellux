//go:build linux || (darwin && cgo)

package systeminfo

import (
	"testing"
	"time"
)

func TestNativeCPUUsage(t *testing.T) {
	// Start a fresh timed sample. A preceding CPUInfo test can otherwise leave
	// identical kernel counters when this test runs within the same clock tick.
	cpuUsageState.Lock()
	cpuUsageState.sampled = false
	cpuUsageState.Unlock()
	// Kernel counters can be cached beyond the first 100 ms, especially on a
	// busy macOS host. Verify a real advancing sample within a bounded window.
	deadline := time.Now().Add(2 * time.Second)
	for {
		info, err := CPUInfo()
		if err != nil {
			t.Fatal(err)
		}
		if info.UsageAvailable {
			if info.UsagePercent < 0 || info.UsagePercent > 100 {
				t.Fatalf("native CPU usage out of range: %+v", info)
			}
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("native CPU counters did not advance: %+v", info)
		}
		time.Sleep(100 * time.Millisecond)
	}
}
