package systeminfo

import "testing"

func TestParseLoadAverage(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    float64
		wantErr bool
	}{
		{name: "linux", input: "1.25 0.80 0.50 2/100 1234\n", want: 1.25},
		{name: "darwin", input: "{ 2.50 1.75 1.00 }\n", want: 2.50},
		{name: "darwin german locale", input: "{ 9,93 5,34 4,64 }\n", want: 9.93},
		{name: "empty", input: "", wantErr: true},
		{name: "invalid", input: "unknown", wantErr: true},
		{name: "negative", input: "-1.0", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseLoadAverage(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseLoadAverage() error = %v, wantErr %t", err, tt.wantErr)
			}
			if err == nil && got != tt.want {
				t.Fatalf("parseLoadAverage() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCPUInfo(t *testing.T) {
	info, err := CPUInfo()
	if err != nil {
		t.Fatalf("CPUInfo() error = %v", err)
	}
	if info.Cores < 1 {
		t.Fatalf("CPUInfo() cores = %d, want at least 1", info.Cores)
	}
	if info.LoadAverage < 0 {
		t.Fatalf("CPUInfo() load average = %v, want non-negative", info.LoadAverage)
	}
}
