package systeminfo

import "testing"

func TestParseVolume(t *testing.T) {
	volume, err := parseVolume("Filesystem 1024-blocks Used Available Capacity Mounted on\n/dev/disk3s1 100000 25000 75000 25% /\n")
	if err != nil {
		t.Fatalf("parseVolume() error = %v", err)
	}
	if volume.Total != 100000*1024 || volume.Used != 25000*1024 || !volume.Available {
		t.Fatalf("parseVolume() = %+v", volume)
	}
}
