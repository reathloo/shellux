package systeminfo

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestTopProcessParsing(t *testing.T) {
	top := parseTopProcess(" 1.0 /usr/bin/idle\n 145.2 /Applications/Example App/worker\nNaN broken\nInf broken\n-2 invalid\n999 ps\n")
	if !top.Available || top.Name != "worker" || top.CPUPercent != 145.2 {
		t.Fatalf("top = %+v", top)
	}
	if parseTopProcess("nonsense\nNaN process\n").Available {
		t.Fatal("invalid process values became available")
	}
}

func TestPressureUsesRealSignals(t *testing.T) {
	for _, tc := range []struct{ input, want string }{{"1\n", "Normal"}, {"2", "Warning"}, {"4", "Critical"}, {"0", ""}, {"missing", ""}} {
		if got := parseDarwinMemoryPressure(tc.input); got != tc.want {
			t.Fatalf("pressure %q = %q", tc.input, got)
		}
	}
	for _, tc := range []struct{ input, want string }{
		{"some avg10=12.35 avg60=1.00 total=150\nfull avg10=0.00", "12.3% PSI"},
		{"some avg10=0.00", "0.0% PSI"}, {"some avg10=NaN", ""},
		{"some avg10=101.0", ""}, {"full avg10=2.0", ""}, {"some avg10=-1", ""},
	} {
		if got := parseMemoryPSI(tc.input); got != tc.want {
			t.Fatalf("PSI %q = %q", tc.input, got)
		}
	}
}

func TestDetailCommandCacheAndTimeout(t *testing.T) {
	dir := t.TempDir()
	command := filepath.Join(dir, "detail")
	counter := filepath.Join(dir, "calls")
	if err := os.WriteFile(command, []byte("#!/bin/sh\nprintf x >> '"+counter+"'\nprintf result\n"), 0700); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if got := detailOutput(command); got != "result" {
			t.Fatalf("output = %q", got)
		}
	}
	data, _ := os.ReadFile(counter)
	if string(data) != "x" {
		t.Fatalf("uncached command ran %q", data)
	}
	slow := filepath.Join(dir, "slow")
	if err := os.WriteFile(slow, []byte("#!/bin/sh\nexec sleep 5\n"), 0700); err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	if detailOutput(slow) != "" || time.Since(start) > 3*time.Second {
		t.Fatal("optional command was not bounded")
	}
	start = time.Now()
	if detailOutput(slow) != "" || time.Since(start) > 500*time.Millisecond {
		t.Fatal("failed command was not cached")
	}
}

func TestConnectionAndGatewayParsing(t *testing.T) {
	ports := "Hardware Port: Ethernet Adapter (en3)\nDevice: en3\nHardware Port: Wi-Fi\nDevice: en0\n"
	if parseHardwarePort(ports, "en0") != "Wi-Fi" || parseHardwarePort(ports, "en3") != "Ethernet" || parseHardwarePort(ports, "missing") != "" {
		t.Fatal("hardware-port matching failed")
	}
	if got := parseNetworkName("Current Wi-Fi Network: Home Office", "Current Wi-Fi Network:"); got != "Home Office" {
		t.Fatal(got)
	}
	if got := parseNetworkName("\tSSID: Home Office\n\tfreq: 5240", "SSID:"); got != "Home Office" {
		t.Fatal(got)
	}
	for _, output := range []string{"You are not associated with an AirPort network.", "Current Wi-Fi Network: <redacted>", ""} {
		if parseNetworkName(output, "Current Wi-Fi Network:") != "" {
			t.Fatal("unavailable SSID was displayed")
		}
	}
	routes := "Iface Destination Gateway Flags\nen0 00000000 0101A8C0 0003\n"
	if parseLinuxGateway(routes) != "192.168.1.1" || parseLinuxGateway(strings.ReplaceAll(routes, "0003", "0000")) != "" {
		t.Fatal("default gateway parsing failed")
	}
}

func TestPingLatencyParsing(t *testing.T) {
	for _, tc := range []struct {
		output    string
		want      time.Duration
		available bool
	}{
		{"64 bytes: time=2.345 ms", 2345 * time.Microsecond, true},
		{"64 bytes: time=0.000 ms", 0, true},
		{"64 bytes: time<1 ms", 0, false},
		{"100% packet loss", 0, false}, {"time=NaN ms", 0, false},
	} {
		got, ok := parsePingLatency(tc.output)
		if got != tc.want || ok != tc.available {
			t.Fatalf("ping %q = %v, %v", tc.output, got, ok)
		}
	}
}
