//go:build darwin

package systeminfo

import (
	"fmt"
	"os/exec"
)

func readLoadAverage() (float64, error) {
	output, err := exec.Command("sysctl", "-n", "vm.loadavg").Output()
	if err != nil {
		return 0, fmt.Errorf("read vm.loadavg: %w", err)
	}

	return parseLoadAverage(string(output))
}
