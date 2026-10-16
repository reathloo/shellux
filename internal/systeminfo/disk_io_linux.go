//go:build linux

package systeminfo

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func readDiskCounters() (diskCounters, error) {
	devices, err := os.ReadDir("/sys/block")
	if err != nil {
		return diskCounters{}, err
	}
	var total diskCounters
	count := 0
	for _, device := range devices {
		name := device.Name()
		if strings.HasPrefix(name, "loop") || strings.HasPrefix(name, "ram") || strings.HasPrefix(name, "zram") || strings.HasPrefix(name, "dm-") || strings.HasPrefix(name, "md") {
			continue // Avoid virtual and stacked devices being counted twice.
		}
		fields, err := os.ReadFile(filepath.Join("/sys/block", name, "stat"))
		if err != nil {
			continue
		}
		read, write, err := parseDiskStats(string(fields))
		if err != nil {
			continue
		}
		total.read += read
		total.write += write
		count++
	}
	if count == 0 {
		return diskCounters{}, fmt.Errorf("no block-device counters")
	}
	return total, nil
}

func parseDiskStats(value string) (uint64, uint64, error) {
	fields := strings.Fields(value)
	if len(fields) < 7 {
		return 0, 0, fmt.Errorf("incomplete disk statistics")
	}
	read, err := strconv.ParseUint(fields[2], 10, 64)
	if err != nil {
		return 0, 0, err
	}
	write, err := strconv.ParseUint(fields[6], 10, 64)
	if err != nil {
		return 0, 0, err
	}
	return read * 512, write * 512, nil
}
