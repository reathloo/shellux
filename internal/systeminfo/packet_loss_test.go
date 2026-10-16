package systeminfo

import (
	"testing"
	"time"
)

func TestPacketLossCountsDistinctGatewayProbes(t *testing.T) {
	lossState.Lock()
	lossState.target, lossState.at, lossState.failed = "", time.Time{}, nil
	lossState.Unlock()
	t.Cleanup(func() {
		lossState.Lock()
		lossState.target, lossState.at, lossState.failed = "", time.Time{}, nil
		lossState.Unlock()
	})
	for i := 0; i < 15; i++ {
		probe := Latency{Target: "192.0.2.1", SampledAt: time.Unix(int64(i+1), 0), Available: i%3 != 0}
		got := ObservePacketLoss(probe)
		if duplicate := ObservePacketLoss(probe); duplicate != got {
			t.Fatalf("cached ping counted twice: %+v, %+v", got, duplicate)
		}
	}
	if got := ObservePacketLoss(Latency{Target: "192.0.2.1", SampledAt: time.Unix(15, 0)}); got.Samples != 12 || got.Lost != 4 {
		t.Fatalf("rolling loss = %+v", got)
	}
	if got := ObservePacketLoss(Latency{Target: "192.0.2.2", SampledAt: time.Unix(16, 0), Available: true}); got.Samples != 1 || got.Lost != 0 {
		t.Fatalf("new gateway did not reset samples: %+v", got)
	}
	if got := ObservePacketLoss(Latency{Target: "192.0.2.2", SampledAt: time.Unix(200, 0)}); got.Samples != 1 || got.Lost != 1 {
		t.Fatalf("stale samples were retained: %+v", got)
	}
	if ObservePacketLoss(Latency{}).Available {
		t.Fatal("missing gateway reported packet loss")
	}
}
