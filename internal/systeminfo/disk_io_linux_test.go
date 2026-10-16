//go:build linux

package systeminfo

import "testing"

func TestParseDiskStatsUsesReadAndWriteSectors(t *testing.T) {
	read, write, err := parseDiskStats("1 2 10 4 5 6 20 8 9 10 11")
	if err != nil || read != 5120 || write != 10240 {
		t.Fatalf("disk bytes = %d, %d, %v", read, write, err)
	}
	if _, _, err := parseDiskStats("1 2"); err == nil {
		t.Fatal("accepted incomplete disk stats")
	}
}
