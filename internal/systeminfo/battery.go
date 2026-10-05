package systeminfo

import "time"

// Battery describes an optional battery reading.
type Battery struct {
	Percentage         int
	Charging           bool
	Available          bool
	Source             string
	Remaining          time.Duration
	RemainingAvailable bool
}

// BatteryInfo returns a battery snapshot. Systems without a battery return an
// unavailable value without an error.
func BatteryInfo() (Battery, error) {
	return readBattery()
}
