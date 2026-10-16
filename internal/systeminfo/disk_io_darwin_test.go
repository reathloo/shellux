//go:build darwin && cgo

package systeminfo

import "testing"

func TestNativeDiskIOCounters(t *testing.T) {
	counters, err := readDiskCounters()
	if err != nil {
		t.Logf("disk I/O counters unavailable on this host: %v", err)
		return
	}
	if counters.read == 0 && counters.write == 0 {
		t.Fatal("physical disk counters are both zero")
	}
	t.Logf("disk bytes: read=%d write=%d", counters.read, counters.write)
}
