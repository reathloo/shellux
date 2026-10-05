//go:build !darwin && !linux

package systeminfo

import "fmt"

func readMemory() (Memory, error) {
	return Memory{}, fmt.Errorf("memory information is not supported on this operating system")
}
