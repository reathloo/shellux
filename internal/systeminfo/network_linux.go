//go:build linux

package systeminfo

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func readNetworkCounters() (networkCounters, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return networkCounters{}, err
	}
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 || len(iface.HardwareAddr) == 0 {
			continue
		}
		if _, err := os.Stat(filepath.Join("/sys/class/net", iface.Name)); err != nil {
			continue
		}
		received, err := readNetworkCounter(iface.Name, "rx_bytes")
		if err != nil {
			continue
		}
		sent, err := readNetworkCounter(iface.Name, "tx_bytes")
		if err != nil {
			continue
		}
		return networkCounters{name: iface.Name, received: received, sent: sent}, nil
	}
	return networkCounters{}, fmt.Errorf("no active network interface")
}

func readNetworkCounter(name, counter string) (uint64, error) {
	value, err := os.ReadFile(filepath.Join("/sys/class/net", name, "statistics", counter))
	if err != nil {
		return 0, err
	}
	parsed, err := strconv.ParseUint(strings.TrimSpace(string(value)), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse network counter: %w", err)
	}
	return parsed, nil
}
