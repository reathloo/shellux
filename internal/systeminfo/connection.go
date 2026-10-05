package systeminfo

import (
	"net"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type Latency struct {
	Target    string
	Duration  time.Duration
	Available bool
}

func connectionDetails(iface string, wantName bool) (kind, name string) {
	if iface == "" {
		return "", ""
	}
	switch runtime.GOOS {
	case "darwin":
		kind = parseHardwarePort(detailOutput("networksetup", "-listallhardwareports"), iface)
		if kind == "Wi-Fi" && wantName {
			name = parseNetworkName(detailOutput("networksetup", "-getairportnetwork", iface), "Current Wi-Fi Network:")
		}
	case "linux":
		if _, err := os.Stat(filepath.Join("/sys/class/net", iface, "wireless")); err == nil {
			kind = "Wi-Fi"
			if wantName {
				name = parseNetworkName(detailOutput("iw", "dev", iface, "link"), "SSID:")
			}
		} else {
			kind = "Ethernet"
		}
	}
	return kind, name
}

func parseHardwarePort(output, iface string) string {
	port := ""
	for _, line := range strings.Split(output, "\n") {
		if value, ok := strings.CutPrefix(line, "Hardware Port:"); ok {
			port = strings.TrimSpace(value)
		}
		if value, ok := strings.CutPrefix(line, "Device:"); ok && strings.TrimSpace(value) == iface {
			if strings.Contains(port, "Wi-Fi") || strings.Contains(port, "AirPort") {
				return "Wi-Fi"
			}
			if strings.Contains(port, "Ethernet") {
				return "Ethernet"
			}
			return port
		}
	}
	return ""
}

func parseNetworkName(output, prefix string) string {
	for _, line := range strings.Split(output, "\n") {
		if value, ok := strings.CutPrefix(strings.TrimSpace(line), prefix); ok {
			value = strings.TrimSpace(value)
			if strings.EqualFold(value, "<redacted>") {
				return ""
			}
			return value
		}
	}
	return ""
}

func gatewayLatency() Latency {
	var target string
	switch runtime.GOOS {
	case "darwin":
		for _, line := range strings.Split(detailOutput("route", "-n", "get", "default"), "\n") {
			fields := strings.Fields(line)
			if len(fields) == 2 && fields[0] == "gateway:" {
				target = fields[1]
			}
		}
	case "linux":
		output, _ := os.ReadFile("/proc/net/route")
		target = parseLinuxGateway(string(output))
	}
	if net.ParseIP(target) == nil {
		return Latency{}
	}
	args := []string{"-n", "-c", "1", "-W", "1", target}
	if runtime.GOOS == "darwin" {
		args[4] = "1000" // macOS uses milliseconds; Linux uses seconds.
	}
	duration, ok := parsePingLatency(detailOutput("ping", args...))
	return Latency{target, duration, ok}
}

func parseLinuxGateway(output string) string {
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 4 || fields[1] != "00000000" {
			continue
		}
		flags, err := strconv.ParseUint(fields[3], 16, 32)
		if err != nil || flags&3 != 3 {
			continue
		}
		gateway, err := strconv.ParseUint(fields[2], 16, 32)
		if err == nil && gateway != 0 {
			return net.IPv4(byte(gateway), byte(gateway>>8), byte(gateway>>16), byte(gateway>>24)).String()
		}
	}
	return ""
}

var pingTimePattern = regexp.MustCompile(`time([=<])\s*([0-9]+(?:\.[0-9]+)?)\s*ms`)

func parsePingLatency(output string) (time.Duration, bool) {
	match := pingTimePattern.FindStringSubmatch(output)
	if len(match) != 3 || match[1] == "<" {
		// An upper bound (time<1) is not an exact measurement.
		return 0, false
	}
	ms, err := strconv.ParseFloat(match[2], 64)
	if err != nil || ms < 0 || ms > 1200 {
		return 0, false
	}
	return time.Duration(ms * float64(time.Millisecond)), true
}
