//go:build !darwin && !linux

package systeminfo

func readTemperature() (Temperature, error) {
	return Temperature{}, nil
}
