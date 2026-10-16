package systeminfo

import (
	"sync"
	"time"
)

// PacketLoss is the failure share of the latest twelve gateway ICMP probes.
// It does not claim to measure application traffic or internet-wide loss.
type PacketLoss struct {
	Lost, Samples int
	Available     bool
}

var lossState struct {
	sync.Mutex
	target string
	at     time.Time
	failed []bool
}

func ObservePacketLoss(probe Latency) PacketLoss {
	if probe.Target == "" || probe.SampledAt.IsZero() {
		return PacketLoss{}
	}
	lossState.Lock()
	defer lossState.Unlock()
	if lossState.target != probe.Target || (!lossState.at.IsZero() && probe.SampledAt.Sub(lossState.at) > 90*time.Second) {
		lossState.target = probe.Target
		lossState.at = time.Time{}
		lossState.failed = nil
	}
	if !lossState.at.Equal(probe.SampledAt) {
		lossState.at = probe.SampledAt
		lossState.failed = append(lossState.failed, !probe.Available)
		if len(lossState.failed) > 12 {
			lossState.failed = lossState.failed[1:]
		}
	}
	result := PacketLoss{Samples: len(lossState.failed), Available: true}
	for _, failed := range lossState.failed {
		if failed {
			result.Lost++
		}
	}
	return result
}
