package render

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/reathloo/shellux/internal/config"
	"github.com/reathloo/shellux/internal/systeminfo"
	"github.com/reathloo/shellux/internal/theme"
)

func TestHeaderIncludesMetrics(t *testing.T) {
	output := Header(systeminfo.Snapshot{
		System:      systeminfo.System{Platform: "macOS", Architecture: "arm64", Hostname: "Air", Shell: "zsh", Terminal: "Terminal"},
		CPU:         systeminfo.CPU{Cores: 8, LoadAverage: 1.25},
		Memory:      systeminfo.Memory{Total: 100, Available: 25},
		Temperature: systeminfo.Temperature{Available: true, Celsius: 42},
		Battery:     systeminfo.Battery{Available: true, Percentage: 80, Charging: true},
		Playback:    systeminfo.Playback{Available: true, Playing: true, Artist: "Daft Punk", Title: "Get Lucky", Position: 84 * time.Second, Duration: 369 * time.Second},
		Uptime:      systeminfo.Uptime{Available: true, Duration: 2*time.Hour + 5*time.Minute},
		Network:     systeminfo.Network{Connected: true, Interface: "en0", IPAddress: "192.168.1.24", UploadRate: 2 * 1024 * 1024, DownloadRate: 512 * 1024},
		Directory:   "/tmp/projects",
		Now:         time.Date(2026, 9, 18, 17, 42, 31, 0, time.Local),
	})

	for _, expected := range []string{"S H E L L U X", "17:42:31", "SYSTEM", "PLATFORM", "macOS · arm64 · Air", "SHELL", "zsh · Terminal", "UPTIME", "2h 5m", "NETWORK", "Verbunden · en0", "192.168.1.24", "UP 2.0 MB/s DOWN 512 KB/s", "RESOURCES", "CPU", "MEMORY", "75%", "42°C", "80% ⚡", "SESSION", "projects", "SPOTIFY", "Daft Punk — Get Lucky", "1:24", "6:09"} {
		if !strings.Contains(output, expected) {
			t.Fatalf("Header() does not contain %q:\n%s", expected, output)
		}
	}
}

func TestHeaderUsesThermalStateWhenAvailable(t *testing.T) {
	output := Header(systeminfo.Snapshot{
		Temperature: systeminfo.Temperature{Available: true, State: "Nominal"},
	})
	if !strings.Contains(output, "Nominal") {
		t.Fatalf("Header() = %q, want thermal state", output)
	}
}

func TestHeaderCombinesThermalWithBattery(t *testing.T) {
	output := Header(systeminfo.Snapshot{
		Battery:     systeminfo.Battery{Available: true, Percentage: 80, Charging: true},
		Temperature: systeminfo.Temperature{Available: true, State: "Nominal"},
	})
	if !strings.Contains(output, "BATTERY") || !strings.Contains(output, "80% ⚡ · Nominal") {
		t.Fatalf("Header() = %q, want battery and thermal on one line", output)
	}
	if strings.Contains(output, "THERMAL") {
		t.Fatalf("Header() = %q, want no separate thermal row", output)
	}
}

func TestHeaderHidesThermalStateByDefault(t *testing.T) {
	output := HeaderWithConfig(systeminfo.Snapshot{
		Temperature: systeminfo.Temperature{Available: true, State: "Nominal"},
	}, "eye", theme.DarkLuxury(), config.Default().Visible)
	if strings.Contains(output, "THERMAL") || strings.Contains(output, "Nominal") {
		t.Fatalf("HeaderWithConfig() = %q, want thermal state hidden by default", output)
	}
}

func TestPlaybackValue(t *testing.T) {
	playback := systeminfo.Playback{Playing: true, Position: 90 * time.Second, Duration: 180 * time.Second}
	if got, want := playbackValue(playback), "▶ 1:30 █████░░░░░ 3:00"; got != want {
		t.Fatalf("playbackValue() = %q, want %q", got, want)
	}
}

func TestUptimeValue(t *testing.T) {
	if got, want := uptimeValue(systeminfo.Uptime{Available: true, Duration: 49*time.Hour + 7*time.Minute}), "2d 1h 7m"; got != want {
		t.Fatalf("uptimeValue() = %q, want %q", got, want)
	}
	if got, want := uptimeValue(systeminfo.Uptime{}), "N/A"; got != want {
		t.Fatalf("uptimeValue() = %q, want %q", got, want)
	}
}

func TestHeaderShowsSpotifyWhenNoTrackIsPlaying(t *testing.T) {
	output := Header(systeminfo.Snapshot{
		Directory: "/tmp",
	})
	for _, expected := range []string{"SESSION", "SPOTIFY", "No song playing"} {
		if !strings.Contains(output, expected) {
			t.Fatalf("Header() does not contain %q:\n%s", expected, output)
		}
	}
}

func TestMemoryValueIncludesCapacityAndPercentage(t *testing.T) {
	memory := systeminfo.Memory{Total: 16 * 1024 * 1024 * 1024, Available: 4 * 1024 * 1024 * 1024}
	if got, want := memoryValueParts(memory, true, true, true), "██████░░ 12.0 GiB (75%)"; got != want {
		t.Fatalf("memoryValue() = %q, want %q", got, want)
	}
}

func TestVolumeValueIncludesBarAndPercentage(t *testing.T) {
	volume := systeminfo.Volume{Available: true, Total: 16 * 1024 * 1024 * 1024, Used: 12 * 1024 * 1024 * 1024}
	if got, want := volumeValueParts(volume, true, true, true), "██████░░ 12.9 / 17.2 GB (75%)"; got != want {
		t.Fatalf("volumeValue() = %q, want %q", got, want)
	}
}

func TestNetworkValues(t *testing.T) {
	connected := systeminfo.Network{Connected: true, Interface: "en0", IPAddress: "192.168.1.24", UploadRate: 2 * 1024 * 1024, DownloadRate: 512 * 1024}
	if got, want := networkStatusValueParts(connected, true, true, true, false, false), "Verbunden · en0 · 192.168.1.24"; got != want {
		t.Fatalf("networkStatusValueParts() = %q, want %q", got, want)
	}
	if got, want := networkTrafficValue(connected, true, true, false), "UP 2.0 MB/s DOWN 512 KB/s"; got != want {
		t.Fatalf("networkTrafficValue() = %q, want %q", got, want)
	}
	if got, want := networkStatusValueParts(systeminfo.Network{}, true, true, true, false, false), "Nicht verbunden"; got != want {
		t.Fatalf("networkStatusValueParts() disconnected = %q, want %q", got, want)
	}
}

func TestTrafficComponentsRenderIndependently(t *testing.T) {
	network := systeminfo.Network{Connected: true, UploadRate: 2 * 1024 * 1024, DownloadRate: 512 * 1024, Latency: systeminfo.Latency{Available: true, Duration: 4 * time.Millisecond}}
	for mask := 0; mask < 8; mask++ {
		visible := config.Default().Visible
		for key := range visible {
			visible[key] = false
		}
		visible["network-traffic"] = true
		for index, part := range config.DisplayParts("traffic") {
			visible["network-traffic."+part] = mask&(1<<index) != 0
		}
		layout := NewHeaderLayout(systeminfo.Snapshot{Network: network}, "eye", theme.DarkLuxury(), visible)
		output := stripANSI(strings.Join(layout.Dashboard, "\n"))
		for _, tc := range []struct {
			text string
			want bool
		}{
			{"UP 2.0 MB/s", mask&1 != 0}, {"DOWN 512 KB/s", mask&2 != 0}, {"4.0 ms", mask&4 != 0},
			{"NETWORK", mask != 0}, {"TRAFFIC", mask != 0}, {"↳ PING", false}, {"Gateway", false},
		} {
			if strings.Contains(output, tc.text) != tc.want {
				t.Fatalf("mask %d: %q in %q", mask, tc.text, output)
			}
		}
		wantSlots, wantRows := 1, 5
		if mask == 0 {
			wantSlots, wantRows = 0, 1
		}
		wantValue := []string{"", "UP 2.0 MB/s", "DOWN 512 KB/s", "UP 2.0 MB/s DOWN 512 KB/s", "4.0 ms", "UP 2.0 MB/s 4.0 ms", "DOWN 512 KB/s 4.0 ms", "UP 2.0 MB/s DOWN 512 KB/s 4.0 ms"}[mask]
		if got := networkTrafficValue(network, mask&1 != 0, mask&2 != 0, mask&4 != 0); got != wantValue {
			t.Fatalf("mask %d: traffic = %q, want %q", mask, got, wantValue)
		}
		if config.MenuSlots(visible) != wantSlots || len(layout.Dashboard) != wantRows {
			t.Fatalf("mask %d: slots/rows = %d/%d", mask, config.MenuSlots(visible), len(layout.Dashboard))
		}
		if fitted := layout.FitTerminal(80, 5); fitted.Height() > 3 {
			t.Fatal("Ping row violated terminal height limit")
		}
		unavailable := stripANSI(NewHeaderLayout(systeminfo.Snapshot{}, "eye", theme.DarkLuxury(), visible).String())
		for _, tc := range []struct {
			text string
			want bool
		}{
			{"UP 0 B/s", mask&1 != 0}, {"DOWN 0 B/s", mask&2 != 0}, {"N/A", mask&4 != 0},
		} {
			if strings.Contains(unavailable, tc.text) != tc.want {
				t.Fatalf("mask %d unavailable: %q", mask, unavailable)
			}
		}
		visible["network-traffic"] = false
		if got := NewHeaderLayout(systeminfo.Snapshot{Network: network}, "eye", theme.DarkLuxury(), visible); len(got.Dashboard) != 1 {
			t.Fatal("Traffic off left a metric row")
		}
	}
}

func TestRateValueUsesBytesBelowOneKilobyte(t *testing.T) {
	if got, want := rateValue(512), "512 B/s"; got != want {
		t.Fatalf("rateValue() = %q, want %q", got, want)
	}
}

func TestCPUValueIncludesUsageBarAndPercentage(t *testing.T) {
	cpu := systeminfo.CPU{Cores: 8, LoadAverage: 4}
	if got, want := cpuValueParts(cpu, true, true, true), "████░░░░ 4.00 load (50%)"; got != want {
		t.Fatalf("cpuValue() = %q, want %q", got, want)
	}
}

func TestHeaderWithEye(t *testing.T) {
	eye := "  eye line one  \n  eye line two  "
	output := HeaderWithEye(systeminfo.Snapshot{}, eye)
	if !strings.Contains(output, "eye line one") || !strings.Contains(output, "S H E L L U X") {
		t.Fatalf("HeaderWithEye() does not contain eye frame:\n%s", output)
	}
}

func TestRainbowStyleColorsDashboard(t *testing.T) {
	layout := NewHeaderLayout(systeminfo.Snapshot{}, "RAINBOW", theme.Rainbow(), nil)
	output := strings.Join(layout.Dashboard, "\n")
	for _, color := range []string{"\033[38;5;196m", "\033[38;5;214m", "\033[38;5;226m", "\033[38;5;46m", "\033[38;5;51m", "\033[38;5;27m", "\033[38;5;201m"} {
		if !strings.Contains(output, color) {
			t.Fatalf("rainbow animation missing color %q: %q", color, output)
		}
	}
}

func TestStyleDoesNotColorAnimation(t *testing.T) {
	layout := NewHeaderLayout(systeminfo.Snapshot{}, "FRAME", theme.Rainbow(), nil)
	if got, want := layout.EyeRegion()[0], "FRAME\033[0m"; got != want {
		t.Fatalf("EyeRegion() = %q, want %q", got, want)
	}
}

func TestAnimationColorsArePreserved(t *testing.T) {
	const purple = "\033[38;5;141m"
	layout := NewHeaderLayout(systeminfo.Snapshot{}, purple+"FRAME\033[0m", theme.Neon(), nil)
	if output := layout.EyeRegion()[0]; !strings.Contains(output, purple+"FRAME") {
		t.Fatalf("EyeRegion() lost animation color: %q", output)
	}
}

func TestHeaderLayoutFitPreservesColoredAnimation(t *testing.T) {
	const purple = "\033[38;5;141m"
	layout := NewHeaderLayout(systeminfo.Snapshot{}, purple+"FRAME\033[0m", theme.Neon(), nil).Fit(3)
	if got, want := layout.Eye[0], purple+"FRA\033[0m"; got != want {
		t.Fatalf("Eye = %q, want %q", got, want)
	}
}

func TestHeaderLayoutHeightUsesTallerRegion(t *testing.T) {
	layout := NewHeaderLayout(systeminfo.Snapshot{}, "one\ntwo", theme.Theme{}, nil)
	if got, want := layout.Height(), len(layout.Dashboard); got != want {
		t.Fatalf("Height() = %d, want %d", got, want)
	}
}

func TestHeaderLayoutFitHidesDashboardBeforeWrapping(t *testing.T) {
	layout := NewHeaderLayout(systeminfo.Snapshot{}, "1234567890", theme.Theme{}, nil).Fit(40)
	if len(layout.Dashboard) != 0 {
		t.Fatalf("Fit() kept dashboard in narrow terminal")
	}
	if layout.EyeWidth > 40 {
		t.Fatalf("Fit() eye width = %d, want at most 40", layout.EyeWidth)
	}
}

func TestHeaderLayoutFitClipsAnimationToTerminalWidth(t *testing.T) {
	layout := NewHeaderLayout(systeminfo.Snapshot{}, "1234567890\nabcdefghij", theme.Theme{}, nil).Fit(6)
	if got, want := layout.Eye, []string{"123456", "abcdef"}; !slices.Equal(got, want) {
		t.Fatalf("Fit() eye = %q, want %q", got, want)
	}
}

func TestHeaderLayoutFitDoesNotModifyOriginal(t *testing.T) {
	layout := NewHeaderLayout(systeminfo.Snapshot{}, "1234567890", theme.Theme{}, nil)
	layout.Fit(3)
	if got := layout.Eye[0]; got != "1234567890" {
		t.Fatalf("Fit() modified original eye: %q", got)
	}
}

func TestHeaderLayoutFitTerminalKeepsRowsForShell(t *testing.T) {
	layout := NewHeaderLayout(systeminfo.Snapshot{}, "one\ntwo\nthree\nfour", theme.Theme{}, nil).FitTerminal(100, 4)
	if got, want := layout.Height(), 2; got != want {
		t.Fatalf("FitTerminal() height = %d, want %d", got, want)
	}
	if got, want := layout.Eye, []string{"one", "two"}; !slices.Equal(got, want) {
		t.Fatalf("FitTerminal() eye = %q, want %q", got, want)
	}
	if got, want := len(layout.Dashboard), 2; got != want {
		t.Fatalf("FitTerminal() dashboard rows = %d, want %d", got, want)
	}
}

func TestHeaderLayoutFitTerminalHidesHeaderWhenNoScrollAreaExists(t *testing.T) {
	for rows := 0; rows <= 2; rows++ {
		layout := NewHeaderLayout(systeminfo.Snapshot{}, "one", theme.Theme{}, nil).FitTerminal(100, rows)
		if got := layout.Height(); got != 0 {
			t.Fatalf("FitTerminal(100, %d) height = %d, want 0", rows, got)
		}
	}
}

func TestDashboardItemKeepsExternalValuesOnOneLine(t *testing.T) {
	output := dashboardItem(theme.Theme{}, "path", "a\nb\rc\td\x1b[2J\u009be\a")
	if got, want := output, "│  PATH      a b c d [2J e "; got != want {
		t.Fatalf("dashboardItem() = %q, want %q", got, want)
	}
	if got := dashboardItem(theme.Theme{}, "spotify", "Björk — Jóga"); !strings.Contains(got, "Björk — Jóga") {
		t.Fatalf("dashboardItem() changed printable Unicode: %q", got)
	}
}

func TestMenuCanHideEverySection(t *testing.T) {
	visible := config.Default().Visible
	for name := range visible {
		visible[name] = false
	}
	layout := NewHeaderLayout(systeminfo.Snapshot{}, "eye", theme.DarkLuxury(), visible)
	if len(layout.Dashboard) != 1 {
		t.Fatalf("empty menu has %d lines, want title only", len(layout.Dashboard))
	}
	visible["temperature"] = true
	layout = NewHeaderLayout(systeminfo.Snapshot{}, "eye", theme.DarkLuxury(), visible)
	output := layout.String()
	if !strings.Contains(output, "RESOURCES") || !strings.Contains(output, "THERMAL") || strings.Contains(output, "SYSTEM") || strings.Contains(output, "NETWORK") || strings.Contains(output, "SESSION") {
		t.Fatalf("thermal-only menu contains unexpected sections: %q", output)
	}
}

func TestUsageComponentsRenderIndependently(t *testing.T) {
	snapshot := systeminfo.Snapshot{
		CPU:    systeminfo.CPU{Cores: 4, LoadAverage: 2},
		Memory: systeminfo.Memory{Total: 8 * 1024 * 1024 * 1024, Available: 4 * 1024 * 1024 * 1024},
		Volume: systeminfo.Volume{Available: true, Total: 8 * 1024 * 1024 * 1024, Used: 4 * 1024 * 1024 * 1024},
	}
	for _, metric := range []struct{ name, label, detail string }{
		{"cpu", "CPU", "2.00 load"}, {"ram", "MEMORY", "4.0 GiB"}, {"volume", "VOLUME", "4.3 / 8.6 GB"},
	} {
		for mask := 0; mask < 8; mask++ {
			visible := config.Default().Visible
			for key := range visible {
				visible[key] = false
			}
			visible[metric.name] = true
			visible[metric.name+".bar"] = mask&1 != 0
			visible[metric.name+".percent"] = mask&2 != 0
			visible[metric.name+".value"] = mask&4 != 0
			layout := NewHeaderLayout(snapshot, "eye", theme.DarkLuxury(), visible)
			output := stripANSI(strings.Join(layout.Dashboard, "\n"))
			for _, check := range []struct {
				text string
				want bool
			}{
				{"████░░░░", mask&1 != 0}, {"50%", mask&2 != 0}, {metric.detail, mask&4 != 0}, {metric.label, mask != 0}, {"RESOURCES", mask != 0},
			} {
				if strings.Contains(output, check.text) != check.want {
					t.Fatalf("%s mask %d: %q visibility wrong in %q", metric.name, mask, check.text, output)
				}
			}
			visible[metric.name] = false
			output = strings.Join(NewHeaderLayout(snapshot, "eye", theme.DarkLuxury(), visible).Dashboard, "\n")
			if strings.Contains(output, metric.label) {
				t.Fatalf("whole-entry off did not hide %s", metric.name)
			}
		}
	}
}

func TestNewHeaderComponentsAndSharedSlot(t *testing.T) {
	snapshot := systeminfo.Snapshot{
		CPU:         systeminfo.CPU{Cores: 4, LoadAverage: 2, TopProcess: systeminfo.Process{Name: "worker", CPUPercent: 145, Available: true}},
		Memory:      systeminfo.Memory{Total: 8 << 30, Available: 4 << 30, Pressure: "Warning"},
		Battery:     systeminfo.Battery{Available: true, Percentage: 50, Charging: true, Remaining: 3*time.Hour + 25*time.Minute, RemainingAvailable: true},
		Temperature: systeminfo.Temperature{Available: true, State: "Nominal"},
		Network:     systeminfo.Network{Connected: true, Interface: "en0", IPAddress: "192.168.1.2", Connection: "Wi-Fi", Name: "Home", Latency: systeminfo.Latency{Available: true, Duration: 2 * time.Millisecond}},
	}
	for _, tc := range []struct {
		name    string
		details map[string]string
	}{
		{"cpu", map[string]string{"top": "worker (145%)"}},
		{"ram", map[string]string{"pressure": "Warning"}},
		{"battery", map[string]string{"bar": "████░░░░", "percent": "50%", "value": "⚡", "remaining": "~3h 25m"}},
		{"network-status", map[string]string{"value": "Verbunden", "interface": "en0", "ip": "192.168.1.2", "connection": "Wi-Fi", "name": "Home"}},
	} {
		for part, want := range tc.details {
			visible := config.Default().Visible
			for key := range visible {
				visible[key] = false
			}
			visible[tc.name] = true
			for _, p := range config.DisplayParts(tc.name) {
				visible[tc.name+"."+p] = false
			}
			visible[tc.name+"."+part] = true
			output := stripANSI(strings.Join(NewHeaderLayout(snapshot, "eye", theme.DarkLuxury(), visible).Dashboard, "\n"))
			if !strings.Contains(output, want) || config.MenuSlots(visible) != 1 {
				t.Fatalf("%s.%s alone: %q", tc.name, part, output)
			}
			for otherPart, otherText := range tc.details {
				if otherPart != part && strings.Contains(output, otherText) {
					t.Fatalf("disabled %s leaked: %q", otherPart, output)
				}
			}
			visible[tc.name+"."+part] = false
			layout := NewHeaderLayout(snapshot, "eye", theme.DarkLuxury(), visible)
			if len(layout.Dashboard) != 1 || config.MenuSlots(visible) != 0 {
				t.Fatalf("empty category still visible: %v", layout.Dashboard)
			}
		}
	}
	visible := config.Default().Visible
	for key := range visible {
		visible[key] = false
	}
	visible["battery"], visible["temperature"] = true, true
	for _, part := range config.DisplayParts("battery") {
		visible["battery."+part] = false
	}
	output := stripANSI(strings.Join(NewHeaderLayout(snapshot, "eye", theme.DarkLuxury(), visible).Dashboard, "\n"))
	if !strings.Contains(output, "THERMAL") || strings.Contains(output, "BATTERY") || config.MenuSlots(visible) != 1 {
		t.Fatal(output)
	}
	visible["network-traffic"], visible["network-traffic.latency"] = true, true
	output = stripANSI(strings.Join(NewHeaderLayout(snapshot, "eye", theme.DarkLuxury(), visible).Dashboard, "\n"))
	if !strings.Contains(output, "2.0 ms") {
		t.Fatal(output)
	}
}

func TestNewDetailsUnavailableAndCompact(t *testing.T) {
	if got := batteryValueParts(systeminfo.Battery{Available: true, Percentage: 80}, false, false, false, true); got != "N/A" {
		t.Fatal(got)
	}
	if got := memoryWithPressure(systeminfo.Memory{}, false, false, false, true); got != "N/A" {
		t.Fatal(got)
	}
	if got := topProcessValue(systeminfo.Process{}, dashboardWidth-14); got != "N/A" {
		t.Fatal(got)
	}
	if got := latencyValue(systeminfo.Latency{}); got != "N/A" {
		t.Fatal(got)
	}
	process := systeminfo.Process{Name: "long-process-name", CPUPercent: 250, Available: true}
	if got := topProcessValue(process, dashboardWidth-14); visibleWidth(got) > dashboardWidth-14 || !strings.Contains(got, " (250%)") {
		t.Fatal(got)
	}
	if got := networkStatusValueParts(systeminfo.Network{Connected: true}, false, false, false, true, true); got != "N/A · N/A" {
		t.Fatal(got)
	}
}

func TestCPUAndTopProcessHaveSeparateRowsAndOneSlot(t *testing.T) {
	cpu := systeminfo.CPU{Cores: 8, LoadAverage: 3.7, TopProcess: systeminfo.Process{Name: "plugin-container", CPUPercent: 42, Available: true}}
	for mask := range 16 {
		visible := config.Default().Visible
		for key := range visible {
			visible[key] = false
		}
		visible["cpu"] = true
		visible["cpu.bar"], visible["cpu.percent"], visible["cpu.value"], visible["cpu.top"] = mask&1 != 0, mask&2 != 0, mask&4 != 0, mask&8 != 0
		layout := NewHeaderLayout(systeminfo.Snapshot{CPU: cpu}, "eye", theme.DarkLuxury(), visible)
		lines := []string{}
		for _, styled := range layout.Dashboard {
			line := stripANSI(styled)
			if strings.HasPrefix(line, "│") {
				lines = append(lines, line)
			}
			if visibleWidth(line) > dashboardWidth {
				t.Fatalf("header line exceeds width: %q", line)
			}
		}
		wantRows := 0
		if mask&7 != 0 {
			wantRows++
		}
		if mask&8 != 0 {
			wantRows++
		}
		if mask == 8 {
			wantRows = 1
		}
		if len(lines) != wantRows {
			t.Fatalf("mask %d: %d rows, want %d: %v", mask, len(lines), wantRows, lines)
		}
		wantSlots := 0
		if mask != 0 {
			wantSlots = 1
		}
		if config.MenuSlots(visible) != wantSlots {
			t.Fatalf("mask %d added a menu slot", mask)
		}
		output := strings.Join(lines, "\n")
		for _, tc := range []struct {
			text string
			want bool
		}{
			{"3.70 load", mask&4 != 0}, {"46%", mask&2 != 0}, {"███░░░░░", mask&1 != 0}, {"plugin-container (42%)", mask&8 != 0},
		} {
			if strings.Contains(output, tc.text) != tc.want {
				t.Fatalf("mask %d: %q in %q", mask, tc.text, output)
			}
		}
		if mask&8 != 0 && mask&7 != 0 {
			if strings.Contains(lines[0], "plugin") || !strings.Contains(lines[1], "↳ TOP") {
				t.Fatalf("process not separated: %v", lines)
			}
		}
		fitted := layout.FitTerminal(120, 5)
		if fitted.Height() > 3 {
			t.Fatal("extra CPU row violated the terminal height limit")
		}
	}
}

func TestTopProcessNamesKeepSuffixAndPercentage(t *testing.T) {
	for _, name := range []string{"Firefox", "plugin-container", "very-long-process-name-with-identifying-worker-suffix", "Äpfel-long-process-name-with-über-long-worker-suffix"} {
		for _, width := range []int{32, 38} {
			process := systeminfo.Process{Name: name, CPUPercent: 250, Available: true}
			got := topProcessValue(process, width)
			if visibleWidth(got) > width || !strings.HasSuffix(got, " (250%)") {
				t.Fatalf("lost percentage/width: %q", got)
			}
			if visibleWidth(name+" (250%)") <= width {
				if got != name+" (250%)" {
					t.Fatalf("unnecessarily shortened: %q", got)
				}
			} else if !strings.Contains(got, "…") || !strings.Contains(got, "suffix (250%)") {
				t.Fatalf("lost identifiable ending: %q", got)
			}
		}
	}
}

func TestDateComponentsRenderIndependently(t *testing.T) {
	location, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 9, 27, 12, 0, 0, 0, location)
	for mask := range 8 {
		visible := config.Default().Visible
		for key := range visible {
			visible[key] = false
		}
		visible["date"] = true
		visible["date.value"], visible["date.timezone"], visible["date.utc"] = mask&1 != 0, mask&2 != 0, mask&4 != 0
		snapshot := systeminfo.Snapshot{Now: now, Timezone: "Berlin"}
		output := stripANSI(strings.Join(NewHeaderLayout(snapshot, "eye", theme.DarkLuxury(), visible).Dashboard, "\n"))
		for _, tc := range []struct {
			text string
			want bool
		}{
			{"27.09.2026", mask&1 != 0}, {"Berlin", mask&2 != 0}, {"UTC+02:00", mask&4 != 0}, {"DATE", mask != 0}, {"SYSTEM", mask != 0},
		} {
			if strings.Contains(output, tc.text) != tc.want {
				t.Fatalf("mask %d: %s in %q", mask, tc.text, output)
			}
		}
		wantSlots := 0
		if mask != 0 {
			wantSlots = 1
		}
		if config.MenuSlots(visible) != wantSlots {
			t.Fatalf("mask %d: wrong slot count", mask)
		}
	}
	for _, tc := range []struct {
		now  time.Time
		want string
	}{
		{time.Date(2026, 1, 27, 12, 0, 0, 0, location), "UTC+01:00"},
		{now, "UTC+02:00"},
		{now.UTC(), "UTC+00:00"},
		{now.In(time.FixedZone("Nepal", 5*3600+45*60)), "UTC+05:45"},
		{now.In(time.FixedZone("Newfoundland", -(3*3600 + 30*60))), "UTC-03:30"},
	} {
		if got := dateValueParts(tc.now, "", false, false, true); got != tc.want {
			t.Fatalf("offset = %s, want %s", got, tc.want)
		}
	}
	if got := dateValueParts(now, "", false, true, false); got != "N/A" {
		t.Fatal(got)
	}
}

func TestEmptyCategoriesDisappearAndReturnWithLastEntry(t *testing.T) {
	categories := []struct {
		title string
		names []string
	}{
		{"SYSTEM", []string{"platform", "shell", "uptime"}},
		{"NETWORK", []string{"network-status", "network-traffic"}},
		{"RESOURCES", []string{"cpu", "ram", "volume", "battery", "temperature"}},
		{"SESSION", []string{"directory", "spotify", "progress"}},
	}
	for _, category := range categories {
		t.Run(category.title, func(t *testing.T) {
			t.Setenv("XDG_CONFIG_HOME", t.TempDir())
			initial := NewHeaderLayout(systeminfo.Snapshot{}, "eye", theme.DarkLuxury(), config.Default().Visible)
			for _, name := range category.names {
				if _, err := config.SetDisplay(name, "", false); err != nil {
					t.Fatal(err)
				}
			}
			settings, err := config.LoadDefault()
			if err != nil {
				t.Fatal(err)
			}
			hidden := NewHeaderLayout(systeminfo.Snapshot{}, "eye", theme.DarkLuxury(), settings.Visible)
			output := stripANSI(strings.Join(hidden.Dashboard, "\n"))
			if strings.Contains(output, category.title) {
				t.Fatalf("empty category still visible: %q", output)
			}
			rows := 0
			for _, name := range category.names {
				if config.Default().Visible[name] {
					rows++
				}
			}
			// Each section also occupies its heading, closing frame and separating blank row.
			if len(initial.Dashboard)-len(hidden.Dashboard) != rows+3 {
				t.Fatalf("category left extra rows: before %d, after %d, want %d removed", len(initial.Dashboard), len(hidden.Dashboard), rows+3)
			}
			for _, other := range categories {
				if other.title != category.title && !strings.Contains(output, other.title) {
					t.Fatalf("hiding %s removed %s", category.title, other.title)
				}
			}
			if _, err := config.SetDisplay(category.names[0], "", true); err != nil {
				t.Fatal(err)
			}
			settings, err = config.LoadDefault()
			if err != nil {
				t.Fatal(err)
			}
			restored := NewHeaderLayout(systeminfo.Snapshot{}, "eye", theme.DarkLuxury(), settings.Visible)
			output = stripANSI(strings.Join(restored.Dashboard, "\n"))
			if !strings.Contains(output, category.title) || len(restored.Dashboard) != len(hidden.Dashboard)+4 {
				t.Fatalf("single entry did not restore category and frame: %q", output)
			}
		})
	}
}

func TestAlternativeEntriesUseSnapshotValues(t *testing.T) {
	const gib = 1024 * 1024 * 1024
	snapshot := systeminfo.Snapshot{
		System:  systeminfo.System{Hostname: "dev-machine"},
		CPU:     systeminfo.CPU{Cores: 8},
		Memory:  systeminfo.Memory{Total: 8 * gib, Available: 3 * gib},
		Volume:  systeminfo.Volume{Available: true, Total: 100 * gib, Used: 60 * gib},
		Network: systeminfo.Network{Connected: true, IPAddress: "192.168.1.42", UploadRate: 2048, DownloadRate: 3 * gib / 1024},
		Now:     time.Date(2026, 9, 26, 12, 0, 0, 0, time.UTC),
	}
	for _, entry := range []struct{ name, title, value string }{
		{"hostname", "SYSTEM", "dev-machine"}, {"cores", "RESOURCES", "8"},
		{"ip", "NETWORK", "192.168.1.42"},
		{"date", "SYSTEM", "26.09.2026"},
	} {
		t.Run(entry.name, func(t *testing.T) {
			visible := config.Default().Visible
			for name := range visible {
				visible[name] = false
			}
			visible[entry.name] = true
			layout := NewHeaderLayout(snapshot, "eye", theme.DarkLuxury(), visible)
			output := stripANSI(strings.Join(layout.Dashboard, "\n"))
			if len(layout.Dashboard) != 5 || !strings.Contains(output, entry.title) || !strings.Contains(output, entry.value) {
				t.Fatalf("incorrect alternative output: %q", output)
			}
			if entry.name != "date" {
				output = NewHeaderLayout(systeminfo.Snapshot{}, "eye", theme.DarkLuxury(), visible).String()
				if !strings.Contains(output, "N/A") {
					t.Fatalf("missing unavailable fallback: %q", output)
				}
			}
		})
	}
}

func TestCoresComponentsRenderIndependently(t *testing.T) {
	snapshot := systeminfo.Snapshot{CPU: systeminfo.CPU{Cores: 8, UsagePercent: 50, UsageAvailable: true}}
	for mask := 0; mask < 8; mask++ {
		visible := config.Default().Visible
		for name := range visible {
			visible[name] = false
		}
		visible["cores"] = true
		visible["cores.bar"] = mask&1 != 0
		visible["cores.percent"] = mask&2 != 0
		visible["cores.value"] = mask&4 != 0
		layout := NewHeaderLayout(snapshot, "eye", theme.DarkLuxury(), visible)
		output := stripANSI(strings.Join(layout.Dashboard, "\n"))
		for _, check := range []struct {
			text string
			want bool
		}{
			{"████░░░░", mask&1 != 0}, {"50%", mask&2 != 0}, {"8 cores", mask&4 != 0}, {"CORES", mask != 0}, {"RESOURCES", mask != 0}, {"SYSTEM", false},
		} {
			if strings.Contains(output, check.text) != check.want {
				t.Fatalf("mask %d: visibility of %q wrong in %q", mask, check.text, output)
			}
		}
	}
	if got := coresValue(systeminfo.CPU{Cores: 8}, true, true, true); got != "8 cores N/A" {
		t.Fatalf("missing usage = %q", got)
	}
	if got := coresValue(systeminfo.CPU{Cores: 8}, false, false, true); got != "8 cores" {
		t.Fatalf("count only = %q", got)
	}
}

func TestSpotifyAndProgressCanBeShownIndependently(t *testing.T) {
	for _, available := range []bool{false, true} {
		snapshot := systeminfo.Snapshot{Playback: systeminfo.Playback{Available: available, Playing: true, Artist: "Artist", Title: "Track", Position: time.Minute, Duration: 2 * time.Minute}}
		for mask := 0; mask < 4; mask++ {
			visible := config.Default().Visible
			for name := range visible {
				visible[name] = false
			}
			visible["spotify"] = mask&1 != 0
			visible["progress"] = mask&2 != 0
			layout := NewHeaderLayout(snapshot, "eye", theme.DarkLuxury(), visible)
			output := stripANSI(strings.Join(layout.Dashboard, "\n"))
			for _, check := range []struct {
				text string
				want bool
			}{
				{"SPOTIFY", mask&1 != 0}, {"PROGRESS", mask&2 != 0}, {"SESSION", mask != 0},
				{"Artist — Track", available && mask&1 != 0}, {"1:00", available && mask&2 != 0},
			} {
				if strings.Contains(output, check.text) != check.want {
					t.Fatalf("available=%v mask=%d: %q visibility wrong in %q", available, mask, check.text, output)
				}
			}
			if !available && mask&2 != 0 && !strings.Contains(output, "N/A") {
				t.Fatalf("unavailable progress lacks fallback: %q", output)
			}
		}
	}
}
