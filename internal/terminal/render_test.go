package terminal

import (
	"strings"
	"testing"
)

func TestRegionUsesAbsoluteCursorPositionsWithoutNewlines(t *testing.T) {
	output := Region([]string{"first", "second"}, 2, 4, 20)
	if strings.ContainsAny(output, "\r\n") {
		t.Fatalf("Region() contains a line break: %q", output)
	}
	for _, expected := range []string{"\033[2;4H", "\033[3;4H", "\033[20X", disableWrap, enableWrap} {
		if !strings.Contains(output, expected) {
			t.Fatalf("Region() does not contain %q: %q", expected, output)
		}
	}
}

func TestReserveHeaderCreatesScrollingAreaBelowHeader(t *testing.T) {
	output := ReserveHeader(10)
	if output != "\033[11;r\033[11;1H" {
		t.Fatalf("ReserveHeader() = %q", output)
	}
	if strings.ContainsAny(output, "\r\n") {
		t.Fatalf("ReserveHeader() contains a line break: %q", output)
	}
}

func TestEraseScrollbackOnlyRemovesStoredLines(t *testing.T) {
	if got, want := EraseScrollback(), eraseScrollback; got != want {
		t.Fatalf("EraseScrollback() = %q, want %q", got, want)
	}
}

func TestMaintainAndReleaseHeader(t *testing.T) {
	if got := MaintainHeader(10); got != saveCursor+"\033[11;r"+restoreCursor {
		t.Fatalf("MaintainHeader() = %q", got)
	}
	if got := ReleaseHeader(); got != saveCursor+"\033[r"+restoreCursor {
		t.Fatalf("ReleaseHeader() = %q", got)
	}
}

func TestPinnedRegionsUsesFixedRegionsWithoutLineBreaks(t *testing.T) {
	region := Region([]string{"eye", "feet"}, 1, 1, 10)
	output := PinnedRegions(2, region)
	if strings.ContainsAny(output, "\r\n") {
		t.Fatalf("PinnedRegions() contains a line break: %q", output)
	}
	if !strings.Contains(output, "\033[1;1H") || !strings.Contains(output, "\033[2;1H") {
		t.Fatalf("PinnedRegions() does not contain fixed regions: %q", output)
	}
	if !strings.HasSuffix(output, ReserveHeader(2)) {
		t.Fatalf("PinnedRegions() does not reserve rows: %q", output)
	}
}

func TestPinnedRegionsPreservingContentOnlyClearsReservedRows(t *testing.T) {
	region := Region([]string{"header"}, 1, 1, 10)
	output := PinnedRegionsPreservingContent(2, region)
	if strings.Contains(output, FullScreen("")) {
		t.Fatalf("PinnedRegionsPreservingContent() cleared the terminal: %q", output)
	}
	if got := strings.Count(output, eraseLine); got != 2 {
		t.Fatalf("PinnedRegionsPreservingContent() clears %d rows, want 2", got)
	}
	if !strings.HasSuffix(output, MaintainHeader(2)) {
		t.Fatalf("PinnedRegionsPreservingContent() does not reserve rows: %q", output)
	}
}

func TestClearRowsErasesWithoutLineBreaks(t *testing.T) {
	output := ClearRows(3)
	if strings.ContainsAny(output, "\r\n") {
		t.Fatalf("ClearRows() contains a line break: %q", output)
	}
	if got := strings.Count(output, eraseLine); got != 3 {
		t.Fatalf("ClearRows() erase count = %d, want 3", got)
	}
}

func TestUnpinHeaderErasesFixedRowsBeforeReleasingScrollRegion(t *testing.T) {
	output := UnpinHeader(3)
	if got := strings.Count(output, eraseLine); got != 3 {
		t.Fatalf("UnpinHeader() clears %d rows, want 3", got)
	}
	if !strings.HasSuffix(output, ReleaseHeader()) || strings.LastIndex(output, eraseLine) > strings.Index(output, "\033[r") {
		t.Fatalf("UnpinHeader() releases scrolling before clearing the header: %q", output)
	}
	if strings.Contains(output, FullScreen("")) || strings.Contains(output, eraseScrollback) || strings.ContainsAny(output, "\r\n") {
		t.Fatalf("UnpinHeader() erases command output or changes scrollback: %q", output)
	}
}
