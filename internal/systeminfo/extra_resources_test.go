package systeminfo

import "testing"

func TestExtraResourceParsers(t *testing.T) {
	darwin := parseDarwinSwap("total = 5120.00M  used = 3874.44M  free = 1245.56M  (encrypted)")
	if !darwin.Available || darwin.Total != 5120000000 || darwin.Free != 1245560000 {
		t.Fatalf("macOS swap = %+v", darwin)
	}
	linux := parseLinuxSwap("MemTotal: 8192 kB\nSwapTotal: 4096 kB\nSwapFree: 1024 kB\n")
	if !linux.Available || linux.Total != 4096*1024 || linux.Used() != 3072*1024 {
		t.Fatalf("Linux swap = %+v", linux)
	}
	gpu := parseDarwinGPU("Chipset Model: Apple M1\nType: GPU\nTotal Number of Cores: 7\n")
	if !gpu.Available || gpu.Name != "Apple M1" || gpu.Cores != 7 {
		t.Fatalf("GPU = %+v", gpu)
	}
}
