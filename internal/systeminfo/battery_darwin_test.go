//go:build darwin

package systeminfo

import (
	"testing"
	"time"
)

func TestDarwinBatteryRemaining(t *testing.T) {
	for _, tc := range []struct {
		status    string
		available bool
	}{
		{"discharging; 3:25 remaining", true},
		{"charging; 3:25 remaining", false},
		{"discharging; (no estimate)", false},
		{"discharging; 3:99 remaining", false},
	} {
		battery, err := parseDarwinBattery("Now drawing from 'Battery Power'\n -InternalBattery-0\t87%; " + tc.status + " present: true")
		if err != nil || battery.RemainingAvailable != tc.available {
			t.Fatalf("%s: %+v, %v", tc.status, battery, err)
		}
		if tc.available && battery.Remaining != 3*time.Hour+25*time.Minute {
			t.Fatal(battery.Remaining)
		}
	}
}

func TestParseDarwinBattery(t *testing.T) {
	battery, err := parseDarwinBattery("Now drawing from 'AC Power'\n -InternalBattery-0 (id=123)\t87%; charging;\n")
	if err != nil {
		t.Fatalf("parseDarwinBattery() error = %v", err)
	}
	if !battery.Available || battery.Percentage != 87 || !battery.Charging {
		t.Fatalf("parseDarwinBattery() = %+v, want available 87%% and charging", battery)
	}
}

func TestParseDarwinBatteryOnlyMarksActualCharging(t *testing.T) {
	testCases := []struct {
		name     string
		status   string
		charging bool
	}{
		{name: "charging", status: "charging", charging: true},
		{name: "finishing charge", status: "finishing charge", charging: true},
		{name: "discharging", status: "discharging", charging: false},
		{name: "charged", status: "charged", charging: false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			output := "Now drawing from 'Battery Power'\n -InternalBattery-0 (id=123)\t87%; " + testCase.status + ";\n"
			battery, err := parseDarwinBattery(output)
			if err != nil {
				t.Fatalf("parseDarwinBattery() error = %v", err)
			}
			if battery.Charging != testCase.charging {
				t.Fatalf("Charging = %t, want %t for %q", battery.Charging, testCase.charging, testCase.status)
			}
		})
	}
}
