package systeminfo

import (
	"context"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Optional command-based details are bounded and refreshed at most every five
// seconds, independently of the dashboard's faster metric interval.
var detailCommands = struct {
	sync.Mutex
	values map[string]commandSample
}{values: make(map[string]commandSample)}

type commandSample struct {
	at     time.Time
	output string
}

func detailOutput(name string, args ...string) string {
	key := name + "\x00" + strings.Join(args, "\x00")
	detailCommands.Lock()
	defer detailCommands.Unlock()
	if sample, ok := detailCommands.values[key]; ok && time.Since(sample.at) < 5*time.Second {
		return sample.output
	}
	ctx, cancel := context.WithTimeout(context.Background(), 1200*time.Millisecond)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = append(os.Environ(), "LC_ALL=C")
	cmd.WaitDelay = 100 * time.Millisecond
	output, err := cmd.Output()
	if err != nil {
		output = nil
	}
	detailCommands.values[key] = commandSample{time.Now(), string(output)}
	return string(output)
}

type Process struct {
	Name       string
	CPUPercent float64
	Available  bool
}

func topProcess() Process {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		return Process{}
	}
	return parseTopProcess(detailOutput("ps", "-axo", "pcpu=,comm="))
}

// ps supplies its own CPU average (which can exceed 100% for multithreaded
// processes); this is not the total system utilization shown by cores.
func parseTopProcess(output string) Process {
	var top Process
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		index := strings.IndexAny(line, " \t")
		if index < 0 {
			continue
		}
		percent, err := strconv.ParseFloat(line[:index], 64)
		name := filepath.Base(strings.TrimSpace(line[index:]))
		if err != nil || math.IsNaN(percent) || math.IsInf(percent, 0) || percent < 0 || name == "." || name == "" || name == "ps" {
			continue
		}
		if !top.Available || percent > top.CPUPercent {
			top = Process{name, percent, true}
		}
	}
	return top
}

func memoryPressure() string {
	switch runtime.GOOS {
	case "darwin":
		return parseDarwinMemoryPressure(detailOutput("sysctl", "-n", "kern.memorystatus_vm_pressure_level"))
	case "linux":
		output, err := os.ReadFile("/proc/pressure/memory")
		if err == nil {
			return parseMemoryPSI(string(output))
		}
	}
	return ""
}

func parseDarwinMemoryPressure(output string) string {
	switch strings.TrimSpace(output) {
	case "1":
		return "Normal"
	case "2":
		return "Warning"
	case "4":
		return "Critical"
	}
	return ""
}

// Linux PSI reports time stalled by memory contention, not RAM occupancy.
// Display the kernel's ten-second percentage rather than invent severity levels.
func parseMemoryPSI(output string) string {
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 || fields[0] != "some" {
			continue
		}
		for _, field := range fields[1:] {
			if strings.HasPrefix(field, "avg10=") {
				percent, err := strconv.ParseFloat(strings.TrimPrefix(field, "avg10="), 64)
				if err == nil && !math.IsNaN(percent) && percent >= 0 && percent <= 100 {
					return strconv.FormatFloat(percent, 'f', 1, 64) + "% PSI"
				}
			}
		}
	}
	return ""
}
