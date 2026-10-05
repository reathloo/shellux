package render

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/charmbracelet/x/ansi"

	"github.com/reathloo/shellux/internal/animation"
	"github.com/reathloo/shellux/internal/config"
	"github.com/reathloo/shellux/internal/systeminfo"
	"github.com/reathloo/shellux/internal/theme"
)

const dashboardWidth = 52

// HeaderLayout contains independently updateable regions of the header.
type HeaderLayout struct {
	Eye             []string
	Dashboard       []string
	EyeWidth        int
	DashboardWidth  int
	DashboardColumn int
	colors          theme.Theme
}

// Height returns the number of terminal rows occupied by the header.
func (layout HeaderLayout) Height() int {
	return max(len(layout.Eye), len(layout.Dashboard))
}

// Fit constrains a header to the visible terminal width. The dashboard is
// hidden before it can wrap below the animation.
func (layout HeaderLayout) Fit(columns int) HeaderLayout {
	if columns > 0 && layout.DashboardColumn+layout.DashboardWidth-1 > columns {
		layout.Dashboard = nil
		layout.DashboardWidth = 0
	}
	if columns > 0 && layout.EyeWidth > columns {
		layout.EyeWidth = columns
		layout.Eye = slices.Clone(layout.Eye)
		for index, line := range layout.Eye {
			layout.Eye[index] = truncateStyledLine(line, columns)
		}
	}
	return layout
}

// FitTerminal constrains the header to both terminal dimensions. Two rows are
// always left for the prompt and its scrolling area, so a small terminal never
// receives an invalid scrolling region below the visible screen.
func (layout HeaderLayout) FitTerminal(columns, rows int) HeaderLayout {
	layout = layout.Fit(columns)
	if rows <= 2 {
		layout.Eye = nil
		layout.Dashboard = nil
		layout.EyeWidth = 0
		layout.DashboardWidth = 0
		return layout
	}
	maxHeight := rows - 2
	if len(layout.Eye) > maxHeight {
		layout.Eye = layout.Eye[:maxHeight]
	}
	if len(layout.Dashboard) > maxHeight {
		layout.Dashboard = layout.Dashboard[:maxHeight]
	}
	return layout
}

// Header renders the compact, static Shellux header.
func Header(snapshot systeminfo.Snapshot) string {
	return HeaderWithEye(snapshot, animation.Frame(animation.ShelluxDefault{}, 0))
}

// HeaderWithEye renders the header with the supplied animation frame.
func HeaderWithEye(snapshot systeminfo.Snapshot, eye string) string {
	return HeaderWithTheme(snapshot, eye, theme.DarkLuxury())
}

// HeaderWithTheme renders the header using semantic theme colors.
func HeaderWithTheme(snapshot systeminfo.Snapshot, eye string, colors theme.Theme) string {
	return HeaderWithConfig(snapshot, eye, colors, nil)
}

// HeaderWithConfig renders only the metrics enabled in visible. A nil map
// keeps the original metrics visible and alternatives hidden.
func HeaderWithConfig(snapshot systeminfo.Snapshot, eye string, colors theme.Theme, visible map[string]bool) string {
	return NewHeaderLayout(snapshot, eye, colors, visible).String()
}

// NewHeaderLayout prepares the eye and dashboard as separate terminal regions.
func NewHeaderLayout(snapshot systeminfo.Snapshot, eye string, colors theme.Theme, visible map[string]bool) HeaderLayout {
	show := func(name string) bool {
		return config.Shown(visible, name)
	}

	dashboard := []string{dashboardTitle(colors, snapshot.Now, show("time"))}
	appendSection := func(title string, items []string) {
		if len(items) > 0 {
			dashboard = append(dashboard, "", sectionStart(colors, title))
			dashboard = append(dashboard, items...)
			dashboard = append(dashboard, sectionEnd(colors))
		}
	}
	var items []string
	add := func(name, label, value string) {
		if config.EntryShown(visible, name) {
			items = append(items, dashboardItem(colors, label, value))
		}
	}
	add("platform", "platform", snapshot.System.Platform+" · "+snapshot.System.Architecture+" · "+snapshot.System.Hostname)
	add("shell", "shell", snapshot.System.Shell+" · "+snapshot.System.Terminal)
	add("uptime", "uptime", uptimeValue(snapshot.Uptime))
	add("date", "date", dateValueParts(snapshot.Now, snapshot.Timezone, show("date.value"), show("date.timezone"), show("date.utc")))
	add("hostname", "hostname", availableText(snapshot.System.Hostname))
	appendSection("SYSTEM", items)
	items = nil
	add("network-status", "status", networkStatusValueParts(snapshot.Network, show("network-status.value"), show("network-status.interface"), show("network-status.ip"), show("network-status.connection"), show("network-status.name")))
	add("network-traffic", "traffic", networkTrafficValue(snapshot.Network, show("network-traffic.upload"), show("network-traffic.download"), show("network-traffic.latency")))
	add("ip", "ip", availableText(snapshot.Network.IPAddress))
	appendSection("NETWORK", items)
	items = nil
	if config.EntryShown(visible, "cpu") {
		metricsShown := show("cpu.bar") || show("cpu.percent") || show("cpu.value")
		if metricsShown {
			add("cpu", "cpu", cpuValueParts(snapshot.CPU, show("cpu.bar"), show("cpu.percent"), show("cpu.value")))
		}
		if show("cpu.top") {
			if metricsShown {
				add("cpu", "  ↳ top", topProcessValue(snapshot.CPU.TopProcess, dashboardWidth-14))
			} else {
				add("cpu", "cpu", "Top · "+topProcessValue(snapshot.CPU.TopProcess, dashboardWidth-20))
			}
		}
	}
	add("cores", "cores", coresValue(snapshot.CPU, show("cores.bar"), show("cores.percent"), show("cores.value")))
	add("ram", "memory", memoryWithPressure(snapshot.Memory, show("ram.bar"), show("ram.percent"), show("ram.value"), show("ram.pressure")))
	add("volume", "volume", volumeValueParts(snapshot.Volume, show("volume.bar"), show("volume.percent"), show("volume.value")))
	if config.EntryShown(visible, "battery") {
		battery := batteryValueParts(snapshot.Battery, show("battery.bar"), show("battery.percent"), show("battery.value"), show("battery.remaining"))
		if show("temperature") {
			battery += " · " + temperatureValue(snapshot.Temperature)
		}
		add("battery", "battery", battery)
	} else {
		add("temperature", "thermal", temperatureValue(snapshot.Temperature))
	}
	appendSection("RESOURCES", items)
	items = nil
	add("directory", "path", compactDirectory(snapshot.Directory))
	hasTrack := snapshot.Playback.Available && snapshot.Playback.Title != ""
	spotify := "No song playing"
	progress := "N/A"
	if hasTrack {
		spotify = snapshot.Playback.Artist + " — " + snapshot.Playback.Title
		progress = playbackValue(snapshot.Playback)
	}
	add("spotify", "spotify", spotify)
	add("progress", "progress", progress)
	appendSection("SESSION", items)
	if colors.Rainbow {
		for index, line := range dashboard {
			dashboard[index] = rainbowize(line, index*3)
		}
	}

	eyeLines := strings.Split(eye, "\n")
	eyeWidth := 0
	for _, line := range eyeLines {
		eyeWidth = max(eyeWidth, visibleWidth(line))
	}
	return HeaderLayout{
		Eye:             eyeLines,
		Dashboard:       dashboard,
		EyeWidth:        eyeWidth,
		DashboardWidth:  dashboardWidth,
		DashboardColumn: eyeWidth + 6,
		colors:          colors,
	}
}

func dashboardTitle(colors theme.Theme, now time.Time, showTime bool) string {
	const title = "S H E L L U X"
	if !showTime {
		return colors.Accent + title + colors.Reset
	}
	clock := now.Format("15:04:05")
	padding := max(1, dashboardWidth-utf8.RuneCountInString(title)-utf8.RuneCountInString(clock))
	return colors.Accent + title + colors.Reset + strings.Repeat(" ", padding) + colors.Secondary + clock + colors.Reset
}

func sectionStart(colors theme.Theme, title string) string {
	return fmt.Sprintf("%s╭─ %s%s", colors.Muted, title, colors.Reset)
}

func sectionEnd(colors theme.Theme) string {
	return colors.Muted + "╰" + colors.Reset
}

func dashboardItem(colors theme.Theme, label, value string) string {
	const valueWidth = dashboardWidth - 14
	// Paths and track metadata are plain text, never terminal instructions.
	value = strings.Map(func(character rune) rune {
		if unicode.IsControl(character) {
			return ' '
		}
		return character
	}, value)
	return fmt.Sprintf("%s│  %s%-9s%s %s", colors.Muted, colors.Secondary, strings.ToUpper(label), colors.Reset, shorten(value, valueWidth))
}

// EyeRegion returns animation-owned eye lines for an independent terminal update.
func (layout HeaderLayout) EyeRegion() []string {
	lines := make([]string, len(layout.Eye))
	for index, line := range layout.Eye {
		lines[index] = line + layout.colors.Reset
	}
	return lines
}

// String composes both regions for one-shot output.
func (layout HeaderLayout) String() string {
	height := layout.Height()

	var builder strings.Builder
	for index := 0; index < height; index++ {
		leftLine := ""
		if index < len(layout.Eye) {
			leftLine = layout.Eye[index]
		}
		rightLine := ""
		if index < len(layout.Dashboard) {
			rightLine = layout.Dashboard[index]
		}
		padding := layout.EyeWidth - visibleWidth(leftLine)
		builder.WriteString(leftLine)
		builder.WriteString(strings.Repeat(" ", padding))
		builder.WriteString(layout.colors.Reset)
		if rightLine != "" {
			builder.WriteString("     ")
			builder.WriteString(rightLine)
		}
		builder.WriteByte('\n')
	}
	return builder.String()
}

var rainbowColors = func() []string {
	colors := make([]string, 0, len(theme.RainbowColors()))
	for _, color := range theme.RainbowColors() {
		colors = append(colors, ansi.NewStyle().ForegroundColor(color).String())
	}
	return colors
}()

// rainbowize removes existing ANSI colors and cycles visible characters through
// a bright rainbow. Keeping spaces unstyled preserves the compact layout.
func rainbowize(line string, offset int) string {
	plain := stripANSI(line)
	var builder strings.Builder
	builder.Grow(len(plain) * 2)
	colorIndex := offset
	for _, character := range plain {
		if character == ' ' {
			builder.WriteRune(character)
			continue
		}
		builder.WriteString(rainbowColors[colorIndex%len(rainbowColors)])
		builder.WriteRune(character)
		builder.WriteString("\033[0m")
		colorIndex++
	}
	return builder.String()
}

func stripANSI(value string) string {
	var builder strings.Builder
	for index := 0; index < len(value); {
		if value[index] == '\033' && index+1 < len(value) && value[index+1] == '[' {
			index += 2
			for index < len(value) {
				if value[index] >= '@' && value[index] <= '~' {
					index++
					break
				}
				index++
			}
			continue
		}
		_, size := utf8.DecodeRuneInString(value[index:])
		builder.WriteString(value[index : index+size])
		index += size
	}
	return builder.String()
}

func visibleWidth(value string) int {
	return utf8.RuneCountInString(stripANSI(value))
}

func truncateStyledLine(value string, width int) string {
	if width <= 0 {
		return ""
	}
	var builder strings.Builder
	visible := 0
	sawANSI := false
	for index := 0; index < len(value) && visible < width; {
		if value[index] == '\033' && index+1 < len(value) && value[index+1] == '[' {
			sawANSI = true
			start := index
			index += 2
			for index < len(value) {
				if value[index] >= '@' && value[index] <= '~' {
					index++
					break
				}
				index++
			}
			builder.WriteString(value[start:index])
			continue
		}
		_, size := utf8.DecodeRuneInString(value[index:])
		builder.WriteString(value[index : index+size])
		index += size
		visible++
	}
	if sawANSI {
		return builder.String() + "\033[0m"
	}
	return builder.String()
}

func compactDirectory(directory string) string {
	home, err := os.UserHomeDir()
	if err == nil && directory == home {
		return "~"
	}
	if err == nil && strings.HasPrefix(directory, home+string(filepath.Separator)) {
		return "~" + strings.TrimPrefix(directory, home)
	}
	return filepath.Clean(directory)
}

func truncateRunes(value string, width int) string {
	runes := []rune(value)
	if len(runes) <= width {
		return value
	}
	return string(runes[:width])
}

func shorten(value string, width int) string {
	if width < 4 || utf8.RuneCountInString(value) <= width {
		return value
	}
	return truncateRunes(value, width-3) + "..."
}

func availableText(value string) string {
	if value == "" {
		return "N/A"
	}
	return value
}

func dateValueParts(now time.Time, timezone string, value, zone, utc bool) string {
	var parts []string
	if value {
		parts = append(parts, now.Format("02.01.2006"))
	}
	if zone {
		parts = append(parts, availableText(timezone))
	}
	if utc {
		_, offset := now.Zone()
		sign := "+"
		if offset < 0 {
			sign = "-"
			offset = -offset
		}
		parts = append(parts, fmt.Sprintf("UTC%s%02d:%02d", sign, offset/3600, offset%3600/60))
	}
	return strings.Join(parts, " · ")
}

func memoryPercentage(memory systeminfo.Memory) float64 {
	if memory.Total == 0 {
		return 0
	}
	return float64(memory.Used()) / float64(memory.Total) * 100
}

func memoryValueParts(memory systeminfo.Memory, bar, percent, value bool) string {
	percentage := memoryPercentage(memory)
	if memory.Total == 0 {
		return "N/A"
	}
	return usageValue(percentage, fmt.Sprintf("%.1f GiB", bytesToGiB(memory.Used())), bar, percent, value)
}

func volumeValueParts(volume systeminfo.Volume, bar, percent, value bool) string {
	if !volume.Available || volume.Total == 0 {
		return "N/A"
	}
	percentage := float64(volume.Used) / float64(volume.Total) * 100
	return usageValue(percentage, fmt.Sprintf("%.1f / %.1f GB", float64(volume.Used)/1e9, float64(volume.Total)/1e9), bar, percent, value)
}

func networkStatusValueParts(network systeminfo.Network, status, iface, ip, connection, name bool) string {
	if !network.Connected {
		return "Nicht verbunden"
	}
	var parts []string
	if status {
		parts = append(parts, "Verbunden")
	}
	if connection {
		parts = append(parts, availableText(network.Connection))
	}
	if name {
		parts = append(parts, availableText(network.Name))
	}
	if iface {
		parts = append(parts, availableText(network.Interface))
	}
	if ip && network.IPAddress != "" {
		parts = append(parts, network.IPAddress)
	}
	return strings.Join(parts, " · ")
}

func latencyValue(latency systeminfo.Latency) string {
	if !latency.Available {
		return "N/A"
	}
	return fmt.Sprintf("%.1f ms", float64(latency.Duration)/float64(time.Millisecond))
}

func topProcessValue(process systeminfo.Process, width int) string {
	if !process.Available {
		return "N/A"
	}
	suffix := fmt.Sprintf(" (%.0f%%)", process.CPUPercent)
	name := []rune(process.Name)
	nameWidth := max(1, width-visibleWidth(suffix))
	if len(name) > nameWidth {
		// Preserve both ends, which often identify the application and worker.
		left := nameWidth / 2
		right := nameWidth - 1 - left
		return string(name[:left]) + "…" + string(name[len(name)-right:]) + suffix
	}
	return process.Name + suffix
}

func memoryWithPressure(memory systeminfo.Memory, bar, percent, value, pressure bool) string {
	base := ""
	if bar || percent || value {
		base = memoryValueParts(memory, bar, percent, value)
	}
	if pressure {
		if base != "" {
			base += " · "
		}
		base += availableText(memory.Pressure)
	}
	return base
}

func networkTrafficValue(network systeminfo.Network, upload, download, latency bool) string {
	if !network.Connected {
		network.UploadRate, network.DownloadRate = 0, 0
	}
	var parts []string
	if upload {
		parts = append(parts, "UP "+rateValue(network.UploadRate))
	}
	if download {
		parts = append(parts, "DOWN "+rateValue(network.DownloadRate))
	}
	if latency {
		parts = append(parts, latencyValue(network.Latency))
	}
	return strings.Join(parts, " ")
}

func rateValue(bytesPerSecond uint64) string {
	const megabyte = 1024 * 1024
	if bytesPerSecond >= megabyte {
		return fmt.Sprintf("%.1f MB/s", float64(bytesPerSecond)/megabyte)
	}
	if bytesPerSecond < 1024 {
		return fmt.Sprintf("%d B/s", bytesPerSecond)
	}
	return fmt.Sprintf("%.0f KB/s", float64(bytesPerSecond)/1024)
}

// coresValue shows the logical core count and measured total utilization.
func coresValue(cpu systeminfo.CPU, bar, percent, value bool) string {
	detail := "N/A"
	if cpu.Cores > 0 {
		detail = fmt.Sprintf("%d cores", cpu.Cores)
	}
	if cpu.UsageAvailable {
		return usageValue(cpu.UsagePercent, detail, bar, percent, value)
	}
	var parts []string
	if value {
		parts = append(parts, detail)
	}
	if bar || percent {
		parts = append(parts, "N/A")
	}
	return strings.Join(parts, " ")
}

func cpuValueParts(cpu systeminfo.CPU, bar, percent, value bool) string {
	if cpu.Cores < 1 {
		return "N/A"
	}
	percentage := min(100, cpu.LoadAverage/float64(cpu.Cores)*100)
	return usageValue(percentage, fmt.Sprintf("%.2f load", cpu.LoadAverage), bar, percent, value)
}

// usageValue joins enabled components without leaving empty punctuation.
func usageValue(percentage float64, detail string, bar, percent, value bool) string {
	var parts []string
	if bar {
		parts = append(parts, usageBar(percentage, 8))
	}
	if value {
		parts = append(parts, detail)
	}
	if percent {
		text := fmt.Sprintf("%.0f%%", percentage)
		if value {
			text = "(" + text + ")"
		}
		parts = append(parts, text)
	}
	return strings.Join(parts, " ")
}

func usageBar(percentage float64, width int) string {
	if width < 1 {
		return ""
	}
	filled := int(percentage / 100 * float64(width))
	filled = max(0, min(width, filled))
	return strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
}

func bytesToGiB(value uint64) float64 {
	return float64(value) / (1024 * 1024 * 1024)
}

func temperatureValue(temperature systeminfo.Temperature) string {
	if !temperature.Available {
		return "N/A"
	}
	if temperature.State != "" {
		return temperature.State
	}
	return fmt.Sprintf("%.0f°C", temperature.Celsius)
}

func batteryValueParts(battery systeminfo.Battery, bar, percent, charging, remaining bool) string {
	if !battery.Available {
		return "N/A"
	}
	var parts []string
	if bar {
		parts = append(parts, usageBar(float64(battery.Percentage), 8))
	}
	if percent {
		parts = append(parts, fmt.Sprintf("%d%%", battery.Percentage))
	}
	if charging {
		if battery.Charging {
			parts = append(parts, "⚡")
		} else if !bar && !percent {
			parts = append(parts, "Not charging")
		}
	}
	if remaining {
		text := "N/A"
		if battery.RemainingAvailable {
			minutes := max(1, int(battery.Remaining.Minutes()))
			text = fmt.Sprintf("~%dh %dm", minutes/60, minutes%60)
		}
		parts = append(parts, text)
	}
	return strings.Join(parts, " ")
}

func uptimeValue(uptime systeminfo.Uptime) string {
	if !uptime.Available {
		return "N/A"
	}
	minutes := max(0, int(uptime.Duration.Truncate(time.Minute).Minutes()))
	days := minutes / (24 * 60)
	minutes %= 24 * 60
	hours := minutes / 60
	minutes %= 60
	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, minutes)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}

func playbackValue(playback systeminfo.Playback) string {
	state := "❚❚"
	if playback.Playing {
		state = "▶"
	}
	return fmt.Sprintf("%s %s %s %s", state, formatPlaybackDuration(playback.Position), playbackBar(playback.Position, playback.Duration, 10), formatPlaybackDuration(playback.Duration))
}

func formatPlaybackDuration(value time.Duration) string {
	seconds := max(0, int(value.Round(time.Second).Seconds()))
	return fmt.Sprintf("%d:%02d", seconds/60, seconds%60)
}

func playbackBar(position, duration time.Duration, width int) string {
	if width < 1 || duration <= 0 {
		return ""
	}
	filled := int(float64(position) / float64(duration) * float64(width))
	filled = max(0, min(width, filled))
	return strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
}
