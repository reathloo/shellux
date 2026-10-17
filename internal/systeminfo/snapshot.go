package systeminfo

import (
	"time"

	"github.com/reathloo/shellux/internal/config"
)

// Snapshot contains the data needed by the static header.
type Snapshot struct {
	System      System
	CPU         CPU
	Memory      Memory
	Load        LoadAverages
	Swap        Swap
	GPU         GPU
	Volume      Volume
	DiskIO      DiskIO
	Network     Network
	LoggedUsers LoggedUsers
	Media       MediaActivity
	Temperature Temperature
	Battery     Battery
	Playback    Playback
	Uptime      Uptime
	Directory   string
	Now         time.Time
	Timezone    string
	Trends      Trends
	Coding      Coding
	Remote      RemoteSession
}

// SnapshotInfo collects all header data. Optional sensors remain unavailable
// instead of preventing the rest of the snapshot from being displayed.
func SnapshotInfo() (Snapshot, error) {
	return SnapshotInfoWithDisplay(nil)
}

// Optional details are collected only when the corresponding entry and
// component are enabled, so hidden integrations do not run commands or probes.
func SnapshotInfoWithDisplay(visible map[string]bool) (Snapshot, error) {
	cpu, err := CPUInfo()
	if err != nil {
		return Snapshot{}, err
	}
	memory, err := MemoryInfo()
	if err != nil {
		return Snapshot{}, err
	}
	directory, err := DirectoryInfo()
	if err != nil {
		return Snapshot{}, err
	}
	var temperature Temperature
	if config.EntryShown(visible, "temperature") {
		temperature, _ = TemperatureInfo()
	}
	var battery Battery
	if config.EntryShown(visible, "battery") {
		battery, _ = BatteryInfo()
	}
	var playback Playback
	if config.EntryShown(visible, "spotify") || config.EntryShown(visible, "progress") {
		playback, _ = PlaybackInfo()
	}
	uptime, _ := UptimeInfo()
	network := NetworkInfo()
	var volume Volume
	if config.EntryShown(visible, "volume") {
		volume, _ = VolumeInfo()
	}
	var diskIO DiskIO
	var load LoadAverages
	if config.EntryShown(visible, "load-average") {
		load = LoadAveragesInfo()
	}
	var swap Swap
	if config.EntryShown(visible, "swap") {
		swap = SwapInfo()
	}
	var gpu GPU
	if config.EntryShown(visible, "gpu") {
		gpu = GPUInfo()
	}
	if config.EntryShown(visible, "disk-io") {
		diskIO = DiskIOInfo()
	}
	var loggedUsers LoggedUsers
	if config.EntryShown(visible, "logged-users") {
		loggedUsers = LoggedUsersInfo()
	}
	var media MediaActivity
	if config.EntryShown(visible, "media-activity") {
		media = MediaActivityInfo(config.Shown(visible, "media-activity.microphone"), config.Shown(visible, "media-activity.camera"))
	}
	if config.EntryShown(visible, "cpu") && config.Shown(visible, "cpu.top") {
		cpu.TopProcess = topProcess()
	}
	if config.EntryShown(visible, "ram") && config.Shown(visible, "ram.pressure") {
		memory.Pressure = memoryPressure()
	}
	if config.EntryShown(visible, "network-status") && (config.Shown(visible, "network-status.connection") || config.Shown(visible, "network-status.name")) {
		network.Connection, network.Name = connectionDetails(network.Interface, config.Shown(visible, "network-status.name"))
	}
	coding := CodingInfo(visible, directory)
	remote := RemoteSessionInfo(visible)
	latencyEnabled := config.EntryShown(visible, "network-traffic") && config.Shown(visible, "network-traffic.latency")
	lossEnabled := config.EntryShown(visible, "packet-loss")
	if network.Connected && (latencyEnabled || lossEnabled) {
		probe := gatewayLatency()
		if latencyEnabled {
			network.Latency = probe
		}
		if lossEnabled {
			network.PacketLoss = ObservePacketLoss(probe)
		}
	}
	now := TimeInfo()
	timezone := ""
	if config.EntryShown(visible, "date") && config.Shown(visible, "date.timezone") {
		timezone = TimezoneName(now)
	}

	return Snapshot{
		System:      SystemInfo(),
		CPU:         cpu,
		Memory:      memory,
		Load:        load,
		Swap:        swap,
		GPU:         gpu,
		Volume:      volume,
		DiskIO:      diskIO,
		Network:     network,
		LoggedUsers: loggedUsers,
		Media:       media,
		Temperature: temperature,
		Battery:     battery,
		Playback:    playback,
		Uptime:      uptime,
		Directory:   directory,
		Now:         now,
		Timezone:    timezone,
		Coding:      coding,
		Remote:      remote,
	}, nil
}
