package systeminfo

import "testing"

func TestBatteryInfoDoesNotFailWhenUnavailable(t *testing.T) {
	battery, err := BatteryInfo()
	if err != nil {
		t.Fatalf("BatteryInfo() error = %v", err)
	}
	if !battery.Available && (battery.Percentage != 0 || battery.Charging || battery.Source != "") {
		t.Fatalf("unavailable battery = %+v, want zero value", battery)
	}
}
