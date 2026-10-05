//go:build darwin

package systeminfo

import "testing"

func TestParseVMStatAvailable(t *testing.T) {
	value, err := parseVMStatAvailable("Mach Virtual Memory Statistics: (page size of 4096 bytes)\nPages free: 100.\nPages inactive: 200.\nPages speculative: 50.\nPages purgeable: 25.\n")
	if err != nil {
		t.Fatalf("parseVMStatAvailable() error = %v", err)
	}
	if value == 0 {
		t.Fatal("parseVMStatAvailable() returned zero")
	}
}
