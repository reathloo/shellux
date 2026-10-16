package systeminfo

import (
	"sync"
	"time"
)

// DiskIO reports host block-device throughput since the preceding sample.
// The first sample establishes a baseline and intentionally has zero rates.
type DiskIO struct {
	ReadRate, WriteRate uint64
	Available           bool
}

type diskCounters struct{ read, write uint64 }

var diskState struct {
	sync.Mutex
	last diskCounters
	at   time.Time
}

func DiskIOInfo() DiskIO {
	counters, err := readDiskCounters()
	if err != nil {
		return DiskIO{}
	}
	now := time.Now()
	diskState.Lock()
	defer diskState.Unlock()
	result := DiskIO{Available: true}
	if !diskState.at.IsZero() && counters.read >= diskState.last.read && counters.write >= diskState.last.write {
		if seconds := now.Sub(diskState.at).Seconds(); seconds > 0 {
			result.ReadRate = uint64(float64(counters.read-diskState.last.read) / seconds)
			result.WriteRate = uint64(float64(counters.write-diskState.last.write) / seconds)
		}
	}
	diskState.last, diskState.at = counters, now
	return result
}
