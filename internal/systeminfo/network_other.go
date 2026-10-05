//go:build !darwin && !linux

package systeminfo

import "fmt"

func readNetworkCounters() (networkCounters, error) {
	return networkCounters{}, fmt.Errorf("network information is not supported on this operating system")
}
