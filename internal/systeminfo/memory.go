package systeminfo

// Memory describes physical memory values in bytes.
type Memory struct {
	Total     uint64
	Available uint64
	Pressure  string
}

// Used returns the memory currently unavailable to applications.
func (m Memory) Used() uint64 {
	if m.Available >= m.Total {
		return 0
	}
	return m.Total - m.Available
}

// MemoryInfo returns a snapshot of the host physical memory.
func MemoryInfo() (Memory, error) {
	return readMemory()
}
