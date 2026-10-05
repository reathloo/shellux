package systeminfo

import (
	"bufio"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Volume describes used space on the system volume. On macOS, reclaimable
// space is treated as available when the native capacity API is accessible.
type Volume struct {
	Available bool
	Total     uint64
	Used      uint64
}

const volumeRefreshInterval = 30 * time.Second

var volumeState struct {
	sync.Mutex
	value      Volume
	measuredAt time.Time
}

// VolumeInfo caches capacity queries for 30 seconds to avoid interrupting
// animation with native storage accounting or the diskutil fallback.
func VolumeInfo() (Volume, error) {
	volumeState.Lock()
	defer volumeState.Unlock()

	if !volumeState.measuredAt.IsZero() && time.Since(volumeState.measuredAt) < volumeRefreshInterval {
		return volumeState.value, nil
	}

	volume, err := readVolumeInfo()
	if err != nil {
		return Volume{}, err
	}
	volumeState.value = volume
	volumeState.measuredAt = time.Now()
	return volume, nil
}

// volumeInfoFromDF reads a filesystem's allocated space via df.
func volumeInfoFromDF(path string) (Volume, error) {
	output, err := exec.Command("df", "-k", path).Output()
	if err != nil {
		return Volume{}, fmt.Errorf("read system volume: %w", err)
	}
	return parseVolume(string(output))
}

func parseVolume(output string) (Volume, error) {
	scanner := bufio.NewScanner(strings.NewReader(output))
	var line string
	for scanner.Scan() {
		if strings.TrimSpace(scanner.Text()) != "" {
			line = scanner.Text()
		}
	}
	fields := strings.Fields(line)
	if len(fields) < 5 {
		return Volume{}, fmt.Errorf("parse system volume output")
	}
	totalBlocks, err := strconv.ParseUint(fields[1], 10, 64)
	if err != nil {
		return Volume{}, fmt.Errorf("parse system volume total: %w", err)
	}
	usedBlocks, err := strconv.ParseUint(fields[2], 10, 64)
	if err != nil {
		return Volume{}, fmt.Errorf("parse system volume used: %w", err)
	}
	return Volume{Available: true, Total: totalBlocks * 1024, Used: usedBlocks * 1024}, nil
}
