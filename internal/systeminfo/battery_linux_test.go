//go:build linux

package systeminfo

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLinuxBatteryRemaining(t *testing.T) {
	for _, tc := range []struct {
		files map[string]string
		want  time.Duration
	}{
		{map[string]string{"time_to_empty_now": "3600"}, time.Hour},
		{map[string]string{"energy_now": "20000000", "power_now": "10000000"}, 2 * time.Hour},
		{map[string]string{"charge_now": "1500", "current_now": "1000"}, 90 * time.Minute},
		{map[string]string{"energy_now": "2000", "power_now": "0"}, 0},
		{map[string]string{"energy_now": "NaN", "power_now": "1000"}, 0},
		{map[string]string{"time_to_empty_now": "999999999999999999"}, 0},
		{map[string]string{}, 0},
	} {
		dir := t.TempDir()
		for name, value := range tc.files {
			if err := os.WriteFile(filepath.Join(dir, name), []byte(value), 0600); err != nil {
				t.Fatal(err)
			}
		}
		if got := linuxBatteryRemaining(dir); got != tc.want {
			t.Fatalf("%v = %v, want %v", tc.files, got, tc.want)
		}
	}
}
