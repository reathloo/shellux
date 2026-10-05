//go:build (!darwin && !linux) || (darwin && !cgo)

package systeminfo

import "fmt"

func readCPUTimes() (cpuTimes, error) {
	return cpuTimes{}, fmt.Errorf("CPU utilization is unavailable in this build")
}
