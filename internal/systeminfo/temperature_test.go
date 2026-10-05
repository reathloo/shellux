package systeminfo

import "testing"

func TestTemperatureInfoDoesNotFailWhenUnavailable(t *testing.T) {
	temperature, err := TemperatureInfo()
	if err != nil {
		t.Fatalf("TemperatureInfo() error = %v", err)
	}
	if !temperature.Available && (temperature.Celsius != 0 || temperature.State != "" || temperature.Source != "") {
		t.Fatalf("unavailable temperature = %+v, want zero value", temperature)
	}
}
