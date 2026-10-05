//go:build linux

package systeminfo

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func readMemory() (Memory, error) {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return Memory{}, fmt.Errorf("open /proc/meminfo: %w", err)
	}
	defer file.Close()

	values := make(map[string]uint64)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 2 {
			continue
		}

		value, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			return Memory{}, fmt.Errorf("parse %s: %w", fields[0], err)
		}
		values[strings.TrimSuffix(fields[0], ":")] = value * 1024
	}
	if err := scanner.Err(); err != nil {
		return Memory{}, fmt.Errorf("read /proc/meminfo: %w", err)
	}

	memory := Memory{Total: values["MemTotal"], Available: values["MemAvailable"]}
	if memory.Total == 0 || memory.Available == 0 {
		return Memory{}, fmt.Errorf("/proc/meminfo does not contain usable memory values")
	}
	return memory, nil
}
