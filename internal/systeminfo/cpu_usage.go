package systeminfo

import (
	"sync"
	"time"
)

type cpuTimes struct{ total, idle uint64 }

var cpuUsageState struct {
	sync.Mutex
	previous cpuTimes
	sampled  bool
}

// cpuUsage measures busy CPU time across all logical cores. The first call
// takes a short sample; subsequent calls measure since the previous snapshot.
func cpuUsage() (float64, bool) {
	cpuUsageState.Lock()
	defer cpuUsageState.Unlock()
	current, err := readCPUTimes()
	if err != nil {
		cpuUsageState.sampled = false
		return 0, false
	}
	if !cpuUsageState.sampled {
		cpuUsageState.previous = current
		time.Sleep(100 * time.Millisecond)
		current, err = readCPUTimes()
		if err != nil {
			return 0, false
		}
	}
	usage, available := cpuUsageBetween(cpuUsageState.previous, current)
	cpuUsageState.previous = current
	cpuUsageState.sampled = true
	return usage, available
}

func cpuUsageBetween(previous, current cpuTimes) (float64, bool) {
	if current.total <= previous.total || current.idle < previous.idle {
		return 0, false
	}
	elapsed := current.total - previous.total
	idle := current.idle - previous.idle
	if idle > elapsed {
		return 0, false
	}
	return float64(elapsed-idle) / float64(elapsed) * 100, true
}
