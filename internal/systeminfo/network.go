package systeminfo

import (
	"net"
	"sync"
	"time"
)

// Network contains the current connection state and transfer rates.
type Network struct {
	Connected    bool
	Interface    string
	IPAddress    string
	UploadRate   uint64
	DownloadRate uint64
	Connection   string
	Name         string
	Latency      Latency
}

type networkSample struct {
	name       string
	received   uint64
	sent       uint64
	measuredAt time.Time
}

var networkState struct {
	sync.Mutex
	last networkSample
}

// NetworkInfo returns the active interface and rates since the previous
// sample. The first sample intentionally reports zero rates.
func NetworkInfo() Network {
	sample, err := readNetworkCounters()
	if err != nil {
		return Network{}
	}

	now := time.Now()
	networkState.Lock()
	defer networkState.Unlock()
	result := Network{Connected: true, Interface: sample.name, IPAddress: interfaceIPv4(sample.name)}
	if networkState.last.name == sample.name && !networkState.last.measuredAt.IsZero() {
		seconds := now.Sub(networkState.last.measuredAt).Seconds()
		if seconds > 0 && sample.received >= networkState.last.received && sample.sent >= networkState.last.sent {
			result.DownloadRate = uint64(float64(sample.received-networkState.last.received) / seconds)
			result.UploadRate = uint64(float64(sample.sent-networkState.last.sent) / seconds)
		}
	}
	networkState.last = networkSample{name: sample.name, received: sample.received, sent: sample.sent, measuredAt: now}
	return result
}

func interfaceIPv4(name string) string {
	iface, err := net.InterfaceByName(name)
	if err != nil {
		return ""
	}
	addresses, err := iface.Addrs()
	if err != nil {
		return ""
	}
	for _, address := range addresses {
		ipNet, ok := address.(*net.IPNet)
		if !ok || ipNet.IP.IsLoopback() {
			continue
		}
		if ip := ipNet.IP.To4(); ip != nil {
			return ip.String()
		}
	}
	return ""
}

type networkCounters struct {
	name     string
	received uint64
	sent     uint64
}
