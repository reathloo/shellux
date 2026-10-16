package systeminfo

import (
	"testing"
	"time"
)

func TestTrendRecorderKeepsOneMinuteAndDownsamples(t *testing.T) {
	var recorder TrendRecorder
	start := time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)
	var trends Trends
	for i := 0; i < 70; i++ {
		trends = recorder.Observe(Snapshot{
			Now:     start.Add(time.Duration(i) * time.Second),
			CPU:     CPU{Cores: 1, LoadAverage: float64(i) / 100},
			Memory:  Memory{Total: 100, Available: uint64(100 - i)},
			Network: Network{Connected: true, DownloadRate: uint64(i * 100)},
			Battery: Battery{Available: true, Percentage: 100 - i},
		})
	}
	for name, values := range map[string][]float64{
		"cpu": trends.CPU, "memory": trends.Memory, "network": trends.Network, "battery": trends.Battery,
	} {
		if len(values) != trendWidth {
			t.Fatalf("%s trend has %d values, want %d", name, len(values), trendWidth)
		}
	}
	if trends.CPU[0] < 9 || trends.CPU[len(trends.CPU)-1] < 60 {
		t.Fatalf("old CPU samples were not discarded/downsampled: %v", trends.CPU)
	}
}

func TestTrendRecorderSkipsUnavailableMetrics(t *testing.T) {
	trends := new(TrendRecorder).Observe(Snapshot{Now: time.Now()})
	if len(trends.CPU)+len(trends.Memory)+len(trends.Network)+len(trends.Battery) != 0 {
		t.Fatalf("unavailable metrics produced trends: %+v", trends)
	}
}
