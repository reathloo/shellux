package systeminfo

import "time"

const (
	trendWindow = time.Minute
	trendWidth  = 8
)

// Trends contains compact recent histories prepared for dashboard sparklines.
type Trends struct {
	CPU, Memory, Network, Battery []float64
}

type trendPoint struct {
	at    time.Time
	value float64
}

// TrendRecorder keeps one minute of samples and reduces each metric to eight
// representative values. It intentionally lives only for the running header;
// no monitoring data is written to disk.
type TrendRecorder struct {
	cpu, memory, network, battery []trendPoint
}

// Observe records a snapshot and returns histories ending at that snapshot.
func (r *TrendRecorder) Observe(snapshot Snapshot) Trends {
	now := snapshot.Now
	if now.IsZero() {
		now = time.Now()
	}
	if snapshot.CPU.Cores > 0 {
		usage := snapshot.CPU.LoadAverage / float64(snapshot.CPU.Cores) * 100
		r.cpu = append(r.cpu, trendPoint{now, clampPercent(usage)})
	}
	if snapshot.Memory.Total > 0 {
		usage := float64(snapshot.Memory.Used()) / float64(snapshot.Memory.Total) * 100
		r.memory = append(r.memory, trendPoint{now, clampPercent(usage)})
	}
	if snapshot.Network.Connected {
		traffic := float64(snapshot.Network.UploadRate + snapshot.Network.DownloadRate)
		r.network = append(r.network, trendPoint{now, traffic})
	}
	if snapshot.Battery.Available {
		r.battery = append(r.battery, trendPoint{now, clampPercent(float64(snapshot.Battery.Percentage))})
	}

	cutoff := now.Add(-trendWindow)
	r.cpu = recentPoints(r.cpu, cutoff)
	r.memory = recentPoints(r.memory, cutoff)
	r.network = recentPoints(r.network, cutoff)
	r.battery = recentPoints(r.battery, cutoff)
	return Trends{
		CPU: downsample(r.cpu, trendWidth), Memory: downsample(r.memory, trendWidth),
		Network: downsample(r.network, trendWidth), Battery: downsample(r.battery, trendWidth),
	}
}

func recentPoints(points []trendPoint, cutoff time.Time) []trendPoint {
	first := 0
	for first < len(points) && points[first].at.Before(cutoff) {
		first++
	}
	return append([]trendPoint(nil), points[first:]...)
}

func downsample(points []trendPoint, width int) []float64 {
	if len(points) <= width {
		values := make([]float64, len(points))
		for i, point := range points {
			values[i] = point.value
		}
		return values
	}
	values := make([]float64, width)
	for bucket := range values {
		start := bucket * len(points) / width
		end := (bucket + 1) * len(points) / width
		for i := start; i < end; i++ {
			values[bucket] += points[i].value
		}
		values[bucket] /= float64(end - start)
	}
	return values
}

func clampPercent(value float64) float64 { return max(0, min(100, value)) }
