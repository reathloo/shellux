//go:build darwin

package systeminfo

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// Prefer macOS available capacity, including space the OS can reclaim. The
// diskutil fallback measures physically allocated APFS container space instead.
func readVolumeInfo() (Volume, error) {
	if volume, err := nativeVolumeInfo(); err == nil {
		return volume, nil
	}
	output, err := exec.Command("diskutil", "info", "/").Output()
	if err != nil {
		return Volume{}, fmt.Errorf("read APFS container: %w", err)
	}
	return parseAPFSContainer(string(output))
}

func volumeFromCapacity(total, available int64) (Volume, error) {
	if total <= 0 || available < 0 || available > total {
		return Volume{}, fmt.Errorf("invalid macOS volume capacity: total=%d available=%d", total, available)
	}
	return Volume{Available: true, Total: uint64(total), Used: uint64(total - available)}, nil
}

func parseAPFSContainer(output string) (Volume, error) {
	total, err := apfsContainerBytes(output, "Container Total Space:")
	if err != nil {
		return Volume{}, err
	}
	free, err := apfsContainerBytes(output, "Container Free Space:")
	if err != nil {
		return Volume{}, err
	}
	if free > total {
		return Volume{}, fmt.Errorf("APFS container free space exceeds total space")
	}
	return Volume{Available: true, Total: total, Used: total - free}, nil
}

func apfsContainerBytes(output, prefix string) (uint64, error) {
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, prefix) {
			continue
		}
		start := strings.Index(line, "(")
		end := strings.Index(line, " Bytes)")
		if start < 0 || end <= start+1 {
			break
		}
		value := strings.Fields(line[start+1 : end])
		if len(value) == 0 {
			break
		}
		parsed, err := strconv.ParseUint(value[0], 10, 64)
		if err != nil {
			return 0, fmt.Errorf("parse %s: %w", prefix, err)
		}
		return parsed, nil
	}
	return 0, fmt.Errorf("APFS container output does not contain %s", prefix)
}
