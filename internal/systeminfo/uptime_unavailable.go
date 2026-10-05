//go:build !darwin && !linux

package systeminfo

func readUptime() (Uptime, error) {
	return Uptime{}, nil
}
