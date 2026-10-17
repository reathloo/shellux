package render

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
	"github.com/reathloo/shellux/internal/config"
	"github.com/reathloo/shellux/internal/systeminfo"
	"github.com/reathloo/shellux/internal/theme"
)

func TestFocusClockKeepsStaticCanvasAndDashboardPosition(t *testing.T) {
	colors := theme.Focus()
	first := FocusClock(30 * time.Minute)
	if !strings.Contains(first, "#") || !strings.Contains(first, "\\") || !strings.Contains(first, "_") || strings.Contains(first, "F O K U S") || strings.Contains(first, "remaining") || strings.Contains(first, "verbleibend") {
		t.Fatalf("incomplete focus clock: %q", first)
	}
	for digit, glyph := range focusDigits {
		for row, line := range glyph {
			if len(line) > focusGlyphWidth {
				t.Fatalf("digit %d row %d exceeds %d columns", digit, row, focusGlyphWidth)
			}
		}
	}
	lines := strings.Split(first, "\n")
	if len(lines) != 26 {
		t.Fatalf("focus height = %d", len(lines))
	}
	for row, line := range lines {
		if width := ansi.StringWidth(line); width != 51 {
			t.Fatalf("focus row %d width = %d", row, width)
		}
	}
	for _, shade := range []string{"38;2;91;102;115", "38;2;124;136;149", "38;2;174;186;198", "38;2;210;220;230"} {
		if !strings.Contains(first, shade) {
			t.Fatalf("focus clock lost shade %s", shade)
		}
	}
	c := config.Default()
	if err := c.SetFocusDuration(30); err != nil {
		t.Fatal(err)
	}
	c = c.EffectiveProfile(t.TempDir(), 140, 40)
	layout := NewHeaderLayoutWithConfig(systeminfo.Snapshot{Now: time.Now()}, first, colors, c)
	if layout.EyeWidth != 51 || layout.DashboardColumn != 57 {
		t.Fatalf("focus moved the dashboard: eye %d, dashboard column %d", layout.EyeWidth, layout.DashboardColumn)
	}
	if first == FocusClock(30*time.Minute-time.Second) {
		t.Fatal("clock did not update when the displayed second changed")
	}
}

func TestFocusClockSwitchesUnitsAtOneHour(t *testing.T) {
	for _, tc := range []struct {
		remaining time.Duration
		want      string
	}{
		{0, "00:00"},
		{15*time.Minute + 26*time.Second, "15:26"},
		{30 * time.Minute, "30:00"},
		{time.Hour - time.Nanosecond, "59:59"},
		{time.Hour, "01:00"},
		{time.Hour + time.Minute, "01:01"},
	} {
		if got := focusClockText(tc.remaining); got != tc.want {
			t.Errorf("remaining %s: %q, want %q", tc.remaining, got, tc.want)
		}
	}
}

func TestFocusClockMatchesProvidedFont(t *testing.T) {
	lines := strings.Split(ansi.Strip(FocusClock(15*time.Minute+26*time.Second)), "\n")
	for row := 0; row < focusGlyphHeight; row++ {
		line := lines[9+row]
		for _, glyph := range []struct {
			start int
			want  string
		}{
			{0, focusDigits[1][row]},
			{12, focusDigits[5][row]},
			{29, focusDigits[2][row]},
			{39, focusDigits[6][row]},
		} {
			if got := line[glyph.start : glyph.start+focusGlyphWidth]; got != glyph.want {
				t.Fatalf("glyph row %d at column %d = %q, want %q", row, glyph.start, got, glyph.want)
			}
		}
		if line[10:12] != "  " {
			t.Fatalf("minute digits touch on row %d: %q", row, line)
		}
	}
}

func TestFocusClockStacksBelowMenuOnNarrowTerminal(t *testing.T) {
	c := config.Default()
	if err := c.SetFocusDuration(30); err != nil {
		t.Fatal(err)
	}
	c = c.EffectiveProfile(t.TempDir(), 80, 24)
	layout := NewHeaderLayoutWithConfig(systeminfo.Snapshot{Now: time.Now()}, FocusClock(15*time.Minute+26*time.Second), theme.Focus(), c).FitTerminal(80, 24)
	if !layout.Stacked() || len(layout.Dashboard) != 0 || layout.Height() > 22 {
		t.Fatalf("narrow focus layout did not stack: %+v", layout)
	}
	plain := ansi.Strip(layout.String())
	menuAt := strings.Index(plain, "S H E L L U X")
	clockAt := strings.Index(plain, "######")
	if menuAt < 0 || clockAt <= menuAt {
		t.Fatalf("menu should be above clock:\n%s", plain)
	}
	for _, line := range layout.Eye[len(layout.Eye)-focusGlyphHeight:] {
		if ansi.StringWidth(line) > 80 {
			t.Fatal("stacked clock exceeds terminal width")
		}
		if line[0] != ' ' {
			t.Fatal("stacked clock is not centered")
		}
	}
	wide := NewHeaderLayoutWithConfig(systeminfo.Snapshot{}, FocusClock(15*time.Minute), theme.Focus(), c).FitTerminal(140, 40)
	if wide.Stacked() || wide.DashboardColumn != 57 {
		t.Fatal("wide focus layout should keep side-by-side dashboard")
	}
}

func TestFocusMenuIncludesPlaybackAndProgress(t *testing.T) {
	c := config.Default()
	if err := c.SetFocusDuration(30); err != nil {
		t.Fatal(err)
	}
	c = c.EffectiveProfile(t.TempDir(), 140, 40)
	snapshot := systeminfo.Snapshot{Playback: systeminfo.Playback{
		Available: true, Playing: true, Artist: "Artist", Title: "Track", Position: time.Minute, Duration: 2 * time.Minute,
	}}
	layout := NewHeaderLayoutWithConfig(snapshot, FocusClock(30*time.Minute), theme.Focus(), c)
	dashboard := ansi.Strip(strings.Join(layout.Dashboard, "\n"))
	for _, value := range []string{"SPOTIFY", "Artist — Track", "PROGRESS", "1:00"} {
		if !strings.Contains(dashboard, value) {
			t.Fatalf("Focus dashboard is missing %q: %s", value, dashboard)
		}
	}
}
