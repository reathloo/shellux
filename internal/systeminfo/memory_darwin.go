//go:build darwin

package systeminfo

import (
	"bufio"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

func readMemory() (Memory, error) {
	totalOutput, err := exec.Command("sysctl", "-n", "hw.memsize").Output()
	if err != nil {
		return Memory{}, fmt.Errorf("read hw.memsize: %w", err)
	}
	total, err := strconv.ParseUint(strings.TrimSpace(string(totalOutput)), 10, 64)
	if err != nil {
		return Memory{}, fmt.Errorf("parse hw.memsize: %w", err)
	}

	vmOutput, err := exec.Command("vm_stat").Output()
	if err != nil {
		return Memory{}, fmt.Errorf("read vm_stat: %w", err)
	}
	available, err := parseVMStatAvailable(string(vmOutput))
	if err != nil {
		return Memory{}, err
	}
	return Memory{Total: total, Available: available}, nil
}

func parseVMStatAvailable(value string) (uint64, error) {
	const pageSizePrefix = "page size of "

	pageSize := uint64(0)
	availablePages := uint64(0)
	scanner := bufio.NewScanner(strings.NewReader(value))
	for scanner.Scan() {
		line := scanner.Text()
		if pageSize == 0 {
			if start := strings.Index(line, pageSizePrefix); start >= 0 {
				pageValue := strings.TrimSuffix(strings.TrimSpace(line[start+len(pageSizePrefix):]), " bytes)")
				parsed, err := strconv.ParseUint(pageValue, 10, 64)
				if err != nil {
					return 0, fmt.Errorf("parse vm_stat page size: %w", err)
				}
				pageSize = parsed
			}
		}

		if strings.HasPrefix(line, "Pages free:") ||
			strings.HasPrefix(line, "Pages inactive:") ||
			strings.HasPrefix(line, "Pages speculative:") ||
			strings.HasPrefix(line, "Pages purgeable:") {
			fields := strings.Fields(line)
			if len(fields) < 3 {
				return 0, fmt.Errorf("parse vm_stat line %q", line)
			}
			pages, err := strconv.ParseUint(strings.TrimSuffix(fields[2], "."), 10, 64)
			if err != nil {
				return 0, fmt.Errorf("parse vm_stat pages: %w", err)
			}
			availablePages += pages
		}
	}
	if err := scanner.Err(); err != nil {
		return 0, fmt.Errorf("read vm_stat output: %w", err)
	}
	if pageSize == 0 || availablePages == 0 {
		return 0, fmt.Errorf("vm_stat does not contain usable memory values")
	}

	return availablePages * pageSize, nil
}
