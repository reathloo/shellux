//go:build darwin

package systeminfo

import "testing"

func TestParseAPFSContainer(t *testing.T) {
	output := "Container Total Space:     245.1 GB (245107195904 Bytes)\nContainer Free Space:      69.0 GB (68994170880 Bytes)\n"
	volume, err := parseAPFSContainer(output)
	if err != nil {
		t.Fatalf("parseAPFSContainer() error = %v", err)
	}
	if volume.Total != 245107195904 || volume.Used != 176113025024 {
		t.Fatalf("parseAPFSContainer() = %+v", volume)
	}
}

func TestVolumeFromCapacity(t *testing.T) {
	// Available includes reclaimable space; it must not be counted as used.
	volume, err := volumeFromCapacity(245107195904, 64410000000)
	if err != nil || !volume.Available || volume.Used != 180697195904 || volume.Total != 245107195904 {
		t.Fatalf("volume = %+v, error = %v", volume, err)
	}
	for _, input := range [][2]int64{{0, 0}, {-1, 0}, {100, -1}, {100, 101}} {
		if _, err := volumeFromCapacity(input[0], input[1]); err == nil {
			t.Fatalf("accepted invalid capacities %v", input)
		}
	}
	for _, available := range []int64{0, 100} {
		if v, err := volumeFromCapacity(100, available); err != nil || v.Used != uint64(100-available) {
			t.Fatalf("boundary capacity: %+v, %v", v, err)
		}
	}
}
