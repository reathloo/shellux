package systeminfo

import "testing"

func TestMemoryUsed(t *testing.T) {
	tests := []struct {
		name     string
		memory   Memory
		wantUsed uint64
	}{
		{name: "normal", memory: Memory{Total: 100, Available: 40}, wantUsed: 60},
		{name: "available exceeds total", memory: Memory{Total: 40, Available: 100}, wantUsed: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.memory.Used(); got != tt.wantUsed {
				t.Fatalf("Memory.Used() = %d, want %d", got, tt.wantUsed)
			}
		})
	}
}

func TestMemoryInfo(t *testing.T) {
	memory, err := MemoryInfo()
	if err != nil {
		t.Fatalf("MemoryInfo() error = %v", err)
	}
	if memory.Total == 0 || memory.Available == 0 {
		t.Fatalf("MemoryInfo() = %+v, want non-zero values", memory)
	}
	if memory.Available > memory.Total {
		t.Fatalf("MemoryInfo() available = %d exceeds total = %d", memory.Available, memory.Total)
	}
}
