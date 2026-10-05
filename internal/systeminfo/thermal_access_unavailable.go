//go:build !darwin

package systeminfo

import "fmt"

// EnableThermalAccess is only needed for the macOS powermetrics provider.
func EnableThermalAccess() (bool, error) {
	return false, fmt.Errorf("thermal access setup is only supported on macOS")
}
