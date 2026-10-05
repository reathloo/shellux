//go:build !darwin && !linux

package systeminfo

func readBattery() (Battery, error) {
	return Battery{}, nil
}
