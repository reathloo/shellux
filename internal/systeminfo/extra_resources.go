package systeminfo

import (
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

// LoadAverages are the operating system's runnable-task averages, not CPU percentages.
type LoadAverages struct {
	One, Five, Fifteen float64
	Available          bool
}

func LoadAveragesInfo() LoadAverages {
	var source string
	if runtime.GOOS == "darwin" {
		sample := detailSampleWithin(time.Second, "sysctl", "-n", "vm.loadavg")
		if sample.err != nil {
			return LoadAverages{}
		}
		source = sample.output
	} else if runtime.GOOS == "linux" {
		data, err := os.ReadFile("/proc/loadavg")
		if err != nil {
			return LoadAverages{}
		}
		source = string(data)
	} else {
		return LoadAverages{}
	}
	fields := strings.Fields(strings.Trim(source, "{} \n"))
	if len(fields) < 3 {
		return LoadAverages{}
	}
	values := [3]float64{}
	for i := range values {
		value, err := strconv.ParseFloat(strings.ReplaceAll(fields[i], ",", "."), 64)
		if err != nil || value < 0 {
			return LoadAverages{}
		}
		values[i] = value
	}
	return LoadAverages{values[0], values[1], values[2], true}
}

// Swap reports used and free swap independently from physical RAM.
type Swap struct {
	Total, Free uint64
	Available   bool
}

func SwapInfo() Swap {
	if runtime.GOOS == "darwin" {
		sample := detailSampleWithin(time.Second, "sysctl", "-n", "vm.swapusage")
		if sample.err != nil {
			return Swap{}
		}
		return parseDarwinSwap(sample.output)
	}
	if runtime.GOOS == "linux" {
		data, err := os.ReadFile("/proc/meminfo")
		if err == nil {
			return parseLinuxSwap(string(data))
		}
	}
	return Swap{}
}

func (s Swap) Used() uint64 {
	if s.Free >= s.Total {
		return 0
	}
	return s.Total - s.Free
}

func parseDarwinSwap(source string) Swap {
	var total, free uint64
	fields := strings.Fields(source)
	for i := 0; i+2 < len(fields); i++ {
		if fields[i+1] != "=" {
			continue
		}
		switch fields[i] {
		case "total":
			total = parseSwapSize(fields[i+2])
		case "free":
			free = parseSwapSize(fields[i+2])
		}
	}
	return Swap{total, free, strings.Contains(source, "total =") && strings.Contains(source, "free =")}
}

func parseSwapSize(source string) uint64 {
	source = strings.ReplaceAll(source, ",", ".")
	if len(source) < 2 {
		return 0
	}
	unit := source[len(source)-1]
	value, err := strconv.ParseFloat(source[:len(source)-1], 64)
	if err != nil || value < 0 {
		return 0
	}
	switch unit {
	case 'G':
		return uint64(value * 1e9)
	case 'M':
		return uint64(value * 1e6)
	case 'K':
		return uint64(value * 1e3)
	}
	return 0
}

func parseLinuxSwap(source string) Swap {
	var total, free uint64
	var foundTotal, foundFree bool
	for _, line := range strings.Split(source, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		value, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			continue
		}
		switch fields[0] {
		case "SwapTotal:":
			total, foundTotal = value*1024, true
		case "SwapFree:":
			free, foundFree = value*1024, true
		}
	}
	return Swap{total, free, foundTotal && foundFree}
}

// GPU shows static hardware identity, not an inferred utilization percentage.
type GPU struct {
	Name      string
	Cores     int
	Available bool
}

var gpuCache struct {
	sync.Mutex
	value GPU
	at    time.Time
}

func GPUInfo() GPU {
	gpuCache.Lock()
	defer gpuCache.Unlock()
	if !gpuCache.at.IsZero() && time.Since(gpuCache.at) < 5*time.Minute {
		return gpuCache.value
	}
	var result GPU
	if runtime.GOOS == "darwin" {
		sample := detailSampleWithin(3*time.Second, "system_profiler", "SPDisplaysDataType", "-detailLevel", "mini")
		if sample.err == nil {
			result = parseDarwinGPU(sample.output)
		}
	}
	gpuCache.value, gpuCache.at = result, time.Now()
	return result
}

func parseDarwinGPU(source string) GPU {
	var gpu GPU
	for _, line := range strings.Split(source, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Chipset Model:") {
			if gpu.Name != "" {
				break
			}
			gpu.Name = strings.TrimSpace(strings.TrimPrefix(line, "Chipset Model:"))
		}
		if strings.HasPrefix(line, "Total Number of Cores:") && gpu.Cores == 0 {
			gpu.Cores, _ = strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(line, "Total Number of Cores:")))
		}
	}
	gpu.Available = gpu.Name != ""
	return gpu
}
