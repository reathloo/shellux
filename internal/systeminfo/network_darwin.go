//go:build darwin

package systeminfo

import (
	"bufio"
	"fmt"
	"net"
	"os/exec"
	"strconv"
	"strings"
)

func readNetworkCounters() (networkCounters, error) {
	defaultInterface, err := defaultRouteInterface()
	if err != nil {
		return networkCounters{}, err
	}
	interfaces, err := net.Interfaces()
	if err != nil {
		return networkCounters{}, err
	}
	active := make(map[string]bool)
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp != 0 && iface.Flags&net.FlagLoopback == 0 && len(iface.HardwareAddr) > 0 {
			active[iface.Name] = true
		}
	}
	if !active[defaultInterface] {
		return networkCounters{}, fmt.Errorf("default network interface %q is not active", defaultInterface)
	}

	output, err := exec.Command("netstat", "-ib").Output()
	if err != nil {
		return networkCounters{}, fmt.Errorf("read netstat: %w", err)
	}
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	header := []string{}
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) == 0 {
			continue
		}
		if fields[0] == "Name" {
			header = fields
			continue
		}
		if len(header) == 0 || fields[0] != defaultInterface || len(fields) < len(header) {
			continue
		}
		received, receiveErr := networkField(fields, header, "Ibytes")
		sent, sendErr := networkField(fields, header, "Obytes")
		if receiveErr == nil && sendErr == nil {
			return networkCounters{name: fields[0], received: received, sent: sent}, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return networkCounters{}, err
	}
	return networkCounters{}, fmt.Errorf("no counters for default network interface %q", defaultInterface)
}

func defaultRouteInterface() (string, error) {
	output, err := exec.Command("route", "-n", "get", "default").Output()
	if err != nil {
		return "", fmt.Errorf("read default network route: %w", err)
	}
	name := parseDefaultRouteInterface(string(output))
	if name == "" {
		return "", fmt.Errorf("default network route does not name an interface")
	}
	return name, nil
}

func parseDefaultRouteInterface(output string) string {
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[0] == "interface:" {
			return fields[1]
		}
	}
	return ""
}

func networkField(fields, header []string, name string) (uint64, error) {
	for index, field := range header {
		if field == name && index < len(fields) {
			return strconv.ParseUint(fields[index], 10, 64)
		}
	}
	return 0, fmt.Errorf("network field %q unavailable", name)
}
