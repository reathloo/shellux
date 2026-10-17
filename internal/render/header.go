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
	"github.com/reathloo/shellux/internal/locale"
	"github.com/reathloo/shellux/internal/systeminfo"
	"github.com/reathloo/shellux/internal/theme"
	"github.com/reathloo/shellux/internal/version"
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
	compact         bool
	focus           bool
	stacked         bool
}

// Stacked reports whether the dashboard is above the focus clock, sharing its
// terminal region. Metric updates must repaint that whole region.
func (layout HeaderLayout) Stacked() bool { return layout.stacked }

// Height returns the number of terminal rows occupied by the header.
func (layout HeaderLayout) Height() int {
	return max(len(layout.Eye), len(layout.Dashboard))
}

// Fit constrains a header to the visible terminal width. The dashboard is
// hidden before it can wrap below the animation.
func (layout HeaderLayout) Fit(columns int) HeaderLayout {
	if layout.compact {
		if columns > 0 {
			layout.DashboardWidth = min(dashboardWidth, columns)
			layout.Dashboard = slices.Clone(layout.Dashboard)
			for index, value := range layout.Dashboard {
				layout.Dashboard[index] = truncateStyledLine(value, columns)
			}
		}
		return layout
	}
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
	if layout.focus && columns > 0 && columns < layout.DashboardColumn+layout.DashboardWidth-1 {
		maxHeight := max(0, rows-2)
		menu := make([]string, 0, len(layout.Dashboard))
		for _, line := range layout.Dashboard {
			menu = append(menu, truncateStyledLine(line, columns))
		}
		if len(menu) > maxHeight {
			menu = menu[:maxHeight]
		}
		clock := layout.Eye[9:17] // The eight reference-font rows.
		if len(menu)+1+len(clock) <= maxHeight {
			menu = append(menu, "")
			for _, line := range clock {
				centered := strings.Repeat(" ", max(0, (columns-focusClockWidth)/2)) + line
				menu = append(menu, truncateStyledLine(centered, columns))
			}
		}
		layout.Eye = menu
		layout.EyeWidth = 0
		for _, line := range menu {
			layout.EyeWidth = max(layout.EyeWidth, visibleWidth(line))
		}
		layout.Dashboard = nil
		layout.DashboardWidth = 0
		layout.stacked = true
		return layout
	}
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
func NewHeaderLayout(snapshot systeminfo.Snapshot, eye string, colors theme.Theme, visible map[string]bool, languages ...locale.Language) HeaderLayout {
	settings := config.Default()
	settings.Visible = visible
	return NewHeaderLayoutWithConfig(snapshot, eye, colors, settings, languages...)
}

// NewHeaderLayoutWithConfig also applies the user's group, entry and component
// ordering. NewHeaderLayout remains available for callers that only customize
// visibility.
func NewHeaderLayoutWithConfig(snapshot systeminfo.Snapshot, eye string, colors theme.Theme, settings config.Config, languages ...locale.Language) HeaderLayout {
	lang := locale.English
	if len(languages) > 0 {
		lang = languages[0]
	}
	snapshot.Memory.Pressure = lang.Text(snapshot.Memory.Pressure)
	snapshot.Temperature.State = lang.Text(snapshot.Temperature.State)
	if settings.RuntimeProfile == "minimal" {
		return newMinimalLayout(snapshot, eye, colors, settings, lang)
	}
	show := func(name string) bool {
		return config.Shown(settings.Visible, name)
	}

	dashboard := []string{dashboardTitle(colors, snapshot.Now, show("time"))}
	if settings.RuntimeProfile == "snapshot" {
		dashboard = []string{colors.Accent + "S H E L L U X  ·  SNAPSHOT" + colors.Reset,
			dashboardItem(colors, "captured", snapshot.Now.Format("2006-01-02 15:04:05 -07:00")),
			dashboardItem(colors, "shellux", version.String())}
	}
	appendSection := func(title string, items []string) {
		if len(items) > 0 {
			dashboard = append(dashboard, "", sectionStart(colors, lang.Text(title)))
			dashboard = append(dashboard, items...)
			dashboard = append(dashboard, sectionEnd(colors))
		}
	}
	entries := settings.OrderedMenuEntries()
	for start := 0; start < len(entries); {
		group := entries[start].Group
		end := start
		for end < len(entries) && entries[end].Group == group {
			end++
		}
		if group != "HEADER" {
			var items []string
			for _, entry := range entries[start:end] {
				for _, item := range dashboardEntry(snapshot, settings, entry, lang) {
					items = append(items, dashboardItem(colors, lang.Text(item.label), item.value))
				}
			}
			appendSection(group, items)
		}
		start = end
	}
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
		focus:           settings.RuntimeProfile == "focus",
	}
}

func newMinimalLayout(snapshot systeminfo.Snapshot, pulse string, colors theme.Theme, settings config.Config, lang locale.Language) HeaderLayout {
	title := colors.Accent + pulse + " Shellux" + colors.Reset
	if config.Shown(settings.Visible, "time") {
		title += "  " + colors.Secondary + snapshot.Now.Format("15:04") + colors.Reset
	}
	lines := []string{title}
	for _, entry := range settings.OrderedMenuEntries() {
		if entry.Group == "HEADER" {
			continue
		}
		for _, item := range dashboardEntry(snapshot, settings, entry, lang) {
			lines = append(lines, colors.Secondary+strings.ToUpper(lang.Text(item.label))+colors.Reset+" "+item.value)
		}
	}
	return HeaderLayout{Dashboard: lines, DashboardWidth: dashboardWidth, DashboardColumn: 1, colors: colors, compact: true}
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

type dashboardValue struct{ label, value string }

func enabledParts(settings config.Config, entry config.MenuEntry) []string {
	parts := make([]string, 0, len(entry.Parts))
	for _, part := range entry.Parts {
		if config.Shown(settings.Visible, entry.Key+"."+part) {
			parts = append(parts, part)
		}
	}
	return parts
}

func hasPart(parts []string, wanted string) bool { return slices.Contains(parts, wanted) }

func dashboardLabel(entry config.MenuEntry) string {
	if entry.Key == "ram" {
		return "memory"
	}
	return entry.Name
}

func dashboardEntry(snapshot systeminfo.Snapshot, settings config.Config, entry config.MenuEntry, lang locale.Language) []dashboardValue {
	if !config.EntryShown(settings.Visible, entry.Key) {
		return nil
	}
	parts := enabledParts(settings, entry)
	join := func(values []string, separator string) []dashboardValue {
		return []dashboardValue{{dashboardLabel(entry), strings.Join(values, separator)}}
	}
	switch entry.Key {
	case "remote-identity":
		values := make([]string, 0, len(parts))
		for _, part := range parts {
			if part == "user" {
				values = append(values, availableText(snapshot.Remote.User))
			} else if part == "host" {
				values = append(values, availableText(snapshot.System.Hostname))
			}
		}
		return join(values, "@")
	case "ssh-link":
		values := make([]string, 0, len(parts))
		for _, part := range parts {
			if part == "client" {
				values = append(values, availableText(snapshot.Remote.Client))
			} else if part == "server" {
				values = append(values, availableText(snapshot.Remote.Server))
			}
		}
		return join(values, " → ")
	case "remote-terminal":
		values := make([]string, 0, len(parts))
		for _, part := range parts {
			if part == "tty" {
				values = append(values, "TTY "+availableText(snapshot.Remote.TTY))
			} else if part == "term" {
				values = append(values, availableText(snapshot.Remote.Term))
			}
		}
		return join(values, " · ")
	case "git-branch":
		values := make([]string, 0, len(parts))
		for _, part := range parts {
			switch part {
			case "branch":
				values = append(values, availableText(snapshot.Coding.Branch))
			case "repository":
				values = append(values, "repo "+availableText(snapshot.Coding.Repository))
			}
		}
		return join(values, " · ")
	case "git-sync":
		if !snapshot.Coding.SyncAvailable {
			return join([]string{"N/A"}, "")
		}
		values := make([]string, 0, len(parts))
		for _, part := range parts {
			if part == "ahead" {
				values = append(values, fmt.Sprintf("↑%d", snapshot.Coding.Ahead))
			} else if part == "behind" {
				values = append(values, fmt.Sprintf("↓%d", snapshot.Coding.Behind))
			}
		}
		return join(values, " · ")
	case "runtimes":
		values := make([]string, 0, len(parts))
		for _, part := range parts {
			version := ""
			label := ""
			switch part {
			case "go":
				version = snapshot.Coding.Go
				label = "Go"
			case "node":
				version = snapshot.Coding.Node
				label = "Node"
			case "python":
				version = snapshot.Coding.Python
				label = "Python"
			case "java":
				version = snapshot.Coding.Java
				label = "Java"
			}
			values = append(values, label+" "+availableText(version))
		}
		rows := make([]dashboardValue, 0, (len(values)+1)/2)
		for i := 0; i < len(values); i += 2 {
			label := entry.Name
			if i > 0 {
				label = "  ↳"
			}
			rows = append(rows, dashboardValue{label, strings.Join(values[i:min(i+2, len(values))], " · ")})
		}
		return rows
	case "dev-context":
		values := make([]string, 0, len(parts))
		for _, part := range parts {
			if part == "docker" {
				values = append(values, "Docker "+availableText(snapshot.Coding.Docker))
			} else if part == "kubernetes" {
				values = append(values, "K8s "+availableText(snapshot.Coding.Kubernetes))
			}
		}
		return join(values, " · ")
	case "last-command":
		if !snapshot.Coding.CommandAvailable {
			return join([]string{"N/A"}, "")
		}
		values := make([]string, 0, len(parts))
		for _, part := range parts {
			if part == "duration" {
				elapsed := "<1s"
				if snapshot.Coding.CommandSeconds > 0 {
					elapsed = (time.Duration(snapshot.Coding.CommandSeconds) * time.Second).String()
				}
				values = append(values, elapsed)
			} else if part == "exit" {
				values = append(values, fmt.Sprintf("exit %d", snapshot.Coding.CommandExit))
			}
		}
		return join(values, " · ")
	case "load-average":
		if !snapshot.Load.Available {
			return join([]string{"N/A"}, "")
		}
		values := make([]string, 0, len(parts))
		for _, part := range parts {
			switch part {
			case "one":
				values = append(values, fmt.Sprintf("1m %.2f", snapshot.Load.One))
			case "five":
				values = append(values, fmt.Sprintf("5m %.2f", snapshot.Load.Five))
			case "fifteen":
				values = append(values, fmt.Sprintf("15m %.2f", snapshot.Load.Fifteen))
			}
		}
		return join(values, " · ")
	case "swap":
		if !snapshot.Swap.Available {
			return join([]string{"N/A"}, "")
		}
		values := make([]string, 0, len(parts))
		for _, part := range parts {
			switch part {
			case "value":
				values = append(values, fmt.Sprintf("%.1f / %.1f GB", float64(snapshot.Swap.Used())/1e9, float64(snapshot.Swap.Total)/1e9))
			case "percent":
				if snapshot.Swap.Total > 0 {
					values = append(values, fmt.Sprintf("%.0f%%", 100*float64(snapshot.Swap.Used())/float64(snapshot.Swap.Total)))
				} else {
					values = append(values, "N/A")
				}
			case "remaining":
				values = append(values, fmt.Sprintf("%.1f GB %s", float64(snapshot.Swap.Free)/1e9, lang.Text("free")))
			}
		}
		return join(values, " · ")
	case "gpu":
		values := make([]string, 0, len(parts))
		for _, part := range parts {
			switch part {
			case "name":
				values = append(values, availableText(snapshot.GPU.Name))
			case "cores":
				if snapshot.GPU.Cores > 0 {
					values = append(values, fmt.Sprintf("%d %s", snapshot.GPU.Cores, lang.Text("cores")))
				} else {
					values = append(values, "N/A")
				}
			}
		}
		return join(values, " · ")
	case "platform":
		if settings.RuntimeProfile == "snapshot" {
			return join([]string{snapshot.System.Platform + " · " + snapshot.System.Architecture}, "")
		}
		return join([]string{snapshot.System.Platform + " · " + snapshot.System.Architecture + " · " + snapshot.System.Hostname}, "")
	case "shell":
		return join([]string{snapshot.System.Shell + " · " + snapshot.System.Terminal}, "")
	case "uptime":
		return join([]string{uptimeValue(snapshot.Uptime)}, "")
	case "hostname":
		return join([]string{availableText(snapshot.System.Hostname)}, "")
	case "ip":
		return join([]string{availableText(snapshot.Network.IPAddress)}, "")
	case "disk-io":
		if !snapshot.DiskIO.Available {
			return join([]string{"N/A"}, "")
		}
		values := make([]string, 0, len(parts))
		for _, part := range parts {
			switch part {
			case "read":
				values = append(values, "R "+rateValue(snapshot.DiskIO.ReadRate))
			case "write":
				values = append(values, "W "+rateValue(snapshot.DiskIO.WriteRate))
			}
		}
		return join(values, " · ")
	case "packet-loss":
		loss := snapshot.Network.PacketLoss
		if !loss.Available || loss.Samples == 0 {
			return join([]string{"N/A"}, "")
		}
		values := make([]string, 0, len(parts))
		for _, part := range parts {
			switch part {
			case "percent":
				values = append(values, fmt.Sprintf("%.0f%%", 100*float64(loss.Lost)/float64(loss.Samples)))
			case "samples":
				values = append(values, fmt.Sprintf("%d/%d %s", loss.Lost, loss.Samples, lang.Text("lost")))
			}
		}
		return join(values, " · ")
	case "logged-users":
		if !snapshot.LoggedUsers.Available {
			return join([]string{"N/A"}, "")
		}
		values := make([]string, 0, len(parts))
		for _, part := range parts {
			switch part {
			case "count":
				label := "users"
				if len(snapshot.LoggedUsers.Names) == 1 {
					label = "user"
				}
				values = append(values, fmt.Sprintf("%d %s", len(snapshot.LoggedUsers.Names), lang.Text(label)))
			case "names":
				names := lang.Text("none")
				if len(snapshot.LoggedUsers.Names) > 0 {
					names = strings.Join(snapshot.LoggedUsers.Names, ", ")
				}
				values = append(values, names)
			}
		}
		return join(values, " · ")
	case "media-activity":
		values := make([]string, 0, len(parts))
		for _, part := range parts {
			state := snapshot.Media.Microphone
			label := lang.Text("Mic")
			if part == "camera" {
				state = snapshot.Media.Camera
				label = lang.Text("Camera")
			}
			status := "N/A"
			if state.Available {
				status = lang.Text("Inactive")
				if state.Active {
					status = lang.Text("Active")
				}
			}
			values = append(values, label+" "+status)
		}
		return join(values, " · ")
	case "directory":
		return join([]string{compactDirectory(snapshot.Directory)}, "")
	case "temperature":
		if config.EntryShown(settings.Visible, "battery") {
			return nil
		}
		return join([]string{temperatureValue(snapshot.Temperature)}, "")
	case "spotify":
		value := lang.Text("No song playing")
		if snapshot.Playback.Available && snapshot.Playback.Title != "" {
			value = snapshot.Playback.Artist + " — " + snapshot.Playback.Title
		}
		return join([]string{value}, "")
	case "progress":
		value := "N/A"
		if snapshot.Playback.Available && snapshot.Playback.Title != "" {
			value = playbackValue(snapshot.Playback)
		}
		return join([]string{value}, "")
	case "date":
		values := make([]string, 0, len(parts))
		for _, part := range parts {
			switch part {
			case "value":
				format := "2006-01-02"
				if lang == locale.German {
					format = "02.01.2006"
				}
				values = append(values, snapshot.Now.Format(format))
			case "timezone":
				values = append(values, availableText(snapshot.Timezone))
			case "utc":
				_, offset := snapshot.Now.Zone()
				sign := "+"
				if offset < 0 {
					sign, offset = "-", -offset
				}
				values = append(values, fmt.Sprintf("UTC%s%02d:%02d", sign, offset/3600, offset%3600/60))
			}
		}
		return join(values, " · ")
	case "network-status":
		if !snapshot.Network.Connected {
			return join([]string{lang.Text("Disconnected")}, "")
		}
		values := make([]string, 0, len(parts))
		for _, part := range parts {
			switch part {
			case "value":
				values = append(values, lang.Text("Connected"))
			case "interface":
				values = append(values, availableText(snapshot.Network.Interface))
			case "ip":
				values = append(values, availableText(snapshot.Network.IPAddress))
			case "connection":
				values = append(values, availableText(snapshot.Network.Connection))
			case "name":
				values = append(values, availableText(snapshot.Network.Name))
			}
		}
		return join(values, " · ")
	case "network-traffic":
		network := snapshot.Network
		if !network.Connected {
			network.UploadRate, network.DownloadRate = 0, 0
		}
		values := make([]string, 0, len(parts))
		for _, part := range parts {
			switch part {
			case "upload":
				values = append(values, "UP "+rateValue(network.UploadRate))
			case "download":
				values = append(values, "DOWN "+rateValue(network.DownloadRate))
			case "latency":
				values = append(values, latencyValue(network.Latency))
			case "sparkline":
				values = append(values, sparklineValue(snapshot.Trends.Network, true))
			}
		}
		return join(values, " ")
	case "cpu":
		values := make([]string, 0, len(parts))
		metricsAvailable := snapshot.CPU.Cores > 0
		percentage := 0.0
		if metricsAvailable {
			percentage = min(100, snapshot.CPU.LoadAverage/float64(snapshot.CPU.Cores)*100)
		}
		unavailableAdded := false
		for _, part := range parts {
			switch part {
			case "bar", "percent", "value":
				if !metricsAvailable {
					if !unavailableAdded {
						values, unavailableAdded = append(values, "N/A"), true
					}
					continue
				}
				switch part {
				case "bar":
					values = append(values, usageBar(percentage, 8))
				case "percent":
					percent := fmt.Sprintf("%.0f%%", percentage)
					if hasPart(parts, "value") {
						percent = "(" + percent + ")"
					}
					values = append(values, percent)
				case "value":
					values = append(values, fmt.Sprintf("%.2f %s", snapshot.CPU.LoadAverage, lang.Text("load")))
				}
			case "sparkline":
				values = append(values, sparklineValue(snapshot.Trends.CPU, false))
			}
		}
		rows := []dashboardValue{}
		if len(values) > 0 {
			rows = append(rows, dashboardValue{entry.Name, strings.Join(values, " ")})
		}
		if hasPart(parts, "top") {
			label := "  ↳ top"
			value := topProcessValue(snapshot.CPU.TopProcess, dashboardWidth-14)
			if len(rows) == 0 {
				label, value = entry.Name, "Top · "+topProcessValue(snapshot.CPU.TopProcess, dashboardWidth-20)
			}
			rows = append(rows, dashboardValue{label, value})
		}
		return rows
	case "cores":
		values := make([]string, 0, len(parts))
		usageUnavailableAdded := false
		for _, part := range parts {
			switch part {
			case "bar":
				if snapshot.CPU.UsageAvailable {
					values = append(values, usageBar(snapshot.CPU.UsagePercent, 8))
				} else if !usageUnavailableAdded {
					values, usageUnavailableAdded = append(values, "N/A"), true
				}
			case "percent":
				if snapshot.CPU.UsageAvailable {
					percent := fmt.Sprintf("%.0f%%", snapshot.CPU.UsagePercent)
					if hasPart(parts, "value") {
						percent = "(" + percent + ")"
					}
					values = append(values, percent)
				} else if !usageUnavailableAdded {
					values, usageUnavailableAdded = append(values, "N/A"), true
				}
			case "value":
				value := "N/A"
				if snapshot.CPU.Cores > 0 {
					value = fmt.Sprintf("%d %s", snapshot.CPU.Cores, lang.Text("cores"))
				}
				values = append(values, value)
			}
		}
		return join(values, " ")
	case "ram":
		percentage := memoryPercentage(snapshot.Memory)
		values := make([]string, 0, len(parts))
		unavailableAdded := false
		for _, part := range parts {
			switch part {
			case "bar", "percent", "value":
				if snapshot.Memory.Total == 0 {
					if !unavailableAdded {
						values, unavailableAdded = append(values, "N/A"), true
					}
					continue
				}
				switch part {
				case "bar":
					values = append(values, usageBar(percentage, 8))
				case "percent":
					percent := fmt.Sprintf("%.0f%%", percentage)
					if hasPart(parts, "value") {
						percent = "(" + percent + ")"
					}
					values = append(values, percent)
				case "value":
					values = append(values, fmt.Sprintf("%.1f GiB", bytesToGiB(snapshot.Memory.Used())))
				}
			case "sparkline":
				values = append(values, sparklineValue(snapshot.Trends.Memory, false))
			case "pressure":
				values = append(values, availableText(snapshot.Memory.Pressure))
			}
		}
		return join(values, " · ")
	case "volume":
		percentage := 0.0
		if snapshot.Volume.Total > 0 {
			percentage = float64(snapshot.Volume.Used) / float64(snapshot.Volume.Total) * 100
		}
		free := uint64(0)
		if snapshot.Volume.Used < snapshot.Volume.Total {
			free = snapshot.Volume.Total - snapshot.Volume.Used
		}
		values := make([]string, 0, len(parts))
		if !snapshot.Volume.Available || snapshot.Volume.Total == 0 {
			return join([]string{"N/A"}, "")
		}
		for _, part := range parts {
			switch part {
			case "bar":
				values = append(values, usageBar(percentage, 8))
			case "percent":
				values = append(values, fmt.Sprintf("%.0f%%", percentage))
			case "value":
				if hasPart(parts, "remaining") {
					values = append(values, fmt.Sprintf("%.0f/%.0f GB", float64(snapshot.Volume.Used)/1e9, float64(snapshot.Volume.Total)/1e9))
				} else {
					values = append(values, fmt.Sprintf("%.1f / %.1f GB", float64(snapshot.Volume.Used)/1e9, float64(snapshot.Volume.Total)/1e9))
				}
			case "remaining":
				values = append(values, fmt.Sprintf("%.1f GB %s", float64(free)/1e9, lang.Text("free")))
			}
		}
		return join(values, " ")
	case "battery":
		values := make([]string, 0, len(parts)+1)
		if !snapshot.Battery.Available {
			values = append(values, "N/A")
		} else {
			for _, part := range parts {
				switch part {
				case "bar":
					values = append(values, usageBar(float64(snapshot.Battery.Percentage), 8))
				case "percent":
					values = append(values, fmt.Sprintf("%d%%", snapshot.Battery.Percentage))
				case "value":
					if snapshot.Battery.Charging {
						values = append(values, "⚡")
					} else if !hasPart(parts, "bar") && !hasPart(parts, "percent") {
						values = append(values, lang.Text("Not charging"))
					}
				case "remaining":
					remaining := "N/A"
					if snapshot.Battery.RemainingAvailable {
						minutes := max(1, int(snapshot.Battery.Remaining.Minutes()))
						remaining = fmt.Sprintf("~%dh %dm", minutes/60, minutes%60)
					}
					values = append(values, remaining)
				case "sparkline":
					values = append(values, sparklineValue(snapshot.Trends.Battery, false))
				}
			}
		}
		if config.EntryShown(settings.Visible, "temperature") {
			return []dashboardValue{{dashboardLabel(entry), strings.Join(values, " ") + " · " + temperatureValue(snapshot.Temperature)}}
		}
		return join(values, " ")
	}
	return nil
}

func orderedUsage(parts []string, available bool, percentage float64, detail func() string) []string {
	if !available {
		return []string{"N/A"}
	}
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		switch part {
		case "bar":
			values = append(values, usageBar(percentage, 8))
		case "percent":
			percent := fmt.Sprintf("%.0f%%", percentage)
			if hasPart(parts, "value") {
				percent = "(" + percent + ")"
			}
			values = append(values, percent)
		case "value":
			values = append(values, detail())
		}
	}
	return values
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

func dateValueParts(now time.Time, timezone string, value, zone, utc bool, languages ...locale.Language) string {
	lang := locale.English
	if len(languages) > 0 {
		lang = languages[0]
	}
	var parts []string
	if value {
		format := "2006-01-02"
		if lang == locale.German {
			format = "02.01.2006"
		}
		parts = append(parts, now.Format(format))
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

func networkStatusValueParts(network systeminfo.Network, status, iface, ip, connection, name bool, languages ...locale.Language) string {
	lang := locale.English
	if len(languages) > 0 {
		lang = languages[0]
	}
	if !network.Connected {
		return lang.Text("Disconnected")
	}
	var parts []string
	if status {
		parts = append(parts, lang.Text("Connected"))
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

func memoryWithPressure(memory systeminfo.Memory, trends []float64, bar, sparkline, percent, value, pressure bool) string {
	base := ""
	if bar || percent || value {
		base = memoryValueParts(memory, bar, percent, value)
	}
	base = withSparkline(base, trends, sparkline, false)
	if pressure {
		if base != "" {
			base += " · "
		}
		base += availableText(memory.Pressure)
	}
	return base
}

func networkTrafficValue(network systeminfo.Network, trends []float64, sparkline, upload, download, latency bool) string {
	if !network.Connected {
		network.UploadRate, network.DownloadRate = 0, 0
	}
	var parts []string
	if sparkline {
		parts = append(parts, sparklineValue(trends, true))
	}
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

func withSparkline(value string, trends []float64, enabled, relative bool) string {
	if !enabled {
		return value
	}
	line := sparklineValue(trends, relative)
	if value == "" {
		return line
	}
	return line + " " + value
}

func sparklineValue(values []float64, relative bool) string {
	if len(values) == 0 {
		return "N/A"
	}
	maximum := 100.0
	if relative {
		maximum = 0
		for _, value := range values {
			maximum = max(maximum, value)
		}
		if maximum == 0 {
			maximum = 1
		}
	}
	const levels = "▁▂▃▄▅▆▇█"
	runes := []rune(levels)
	var line strings.Builder
	for _, value := range values {
		index := int(max(0, min(1, value/maximum)) * float64(len(runes)-1))
		line.WriteRune(runes[index])
	}
	return line.String()
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
func coresValue(cpu systeminfo.CPU, bar, percent, value bool, languages ...locale.Language) string {
	lang := locale.English
	if len(languages) > 0 {
		lang = languages[0]
	}
	detail := "N/A"
	if cpu.Cores > 0 {
		detail = fmt.Sprintf("%d %s", cpu.Cores, lang.Text("cores"))
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

func cpuValueParts(cpu systeminfo.CPU, bar, percent, value bool, languages ...locale.Language) string {
	lang := locale.English
	if len(languages) > 0 {
		lang = languages[0]
	}
	if cpu.Cores < 1 {
		return "N/A"
	}
	percentage := min(100, cpu.LoadAverage/float64(cpu.Cores)*100)
	return usageValue(percentage, fmt.Sprintf("%.2f %s", cpu.LoadAverage, lang.Text("load")), bar, percent, value)
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

func batteryValueParts(battery systeminfo.Battery, bar, percent, charging, remaining bool, languages ...locale.Language) string {
	lang := locale.English
	if len(languages) > 0 {
		lang = languages[0]
	}
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
			parts = append(parts, lang.Text("Not charging"))
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
