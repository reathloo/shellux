package systeminfo

import "testing"

func TestCPUUsageBetween(t *testing.T) {
	for _, test := range []struct {
		name          string
		before, after cpuTimes
		want          float64
		available     bool
	}{
		{"half busy", cpuTimes{100, 60}, cpuTimes{200, 110}, 50, true},
		{"idle", cpuTimes{100, 60}, cpuTimes{200, 160}, 0, true},
		{"busy", cpuTimes{100, 60}, cpuTimes{200, 60}, 100, true},
		{"no elapsed time", cpuTimes{100, 60}, cpuTimes{100, 60}, 0, false},
		{"counter reset", cpuTimes{100, 60}, cpuTimes{20, 10}, 0, false},
		{"idle reset", cpuTimes{100, 60}, cpuTimes{200, 50}, 0, false},
		{"invalid delta", cpuTimes{100, 60}, cpuTimes{110, 80}, 0, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, available := cpuUsageBetween(test.before, test.after)
			if got != test.want || available != test.available {
				t.Fatalf("got %v, %v; want %v, %v", got, available, test.want, test.available)
			}
		})
	}
}

func TestParseCPUTimes(t *testing.T) {
	for _, test := range []struct {
		input string
		want  cpuTimes
		valid bool
	}{
		{"cpu 100 20 30 400 50 6 7 8 90 10\ncpu0 1 2 3 4\n", cpuTimes{621, 450}, true},
		{"cpu 10 20 30 40\n", cpuTimes{100, 40}, true},
		{"", cpuTimes{}, false}, {"cpu0 1 2 3 4", cpuTimes{}, false},
		{"cpu 1 2 3", cpuTimes{}, false}, {"cpu 1 invalid 3 4", cpuTimes{}, false},
		{"cpu -1 2 3 4", cpuTimes{}, false},
	} {
		got, err := parseCPUTimes(test.input)
		if (err == nil) != test.valid || (test.valid && got != test.want) {
			t.Fatalf("parseCPUTimes(%q) = %+v, %v", test.input, got, err)
		}
	}
}
