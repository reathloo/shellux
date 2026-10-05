package systeminfo

// Temperature describes an optional temperature reading.
type Temperature struct {
	Celsius   float64
	State     string
	Available bool
	Source    string
}

// TemperatureInfo returns a temperature snapshot. Missing sensors are not an error.
func TemperatureInfo() (Temperature, error) {
	return readTemperature()
}
