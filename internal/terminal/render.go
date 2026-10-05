package terminal

import (
	"fmt"
	"strings"
)

const (
	saveCursor      = "\0337"
	restoreCursor   = "\0338"
	eraseLine       = "\033[2K"
	eraseScrollback = "\033[3J"
	disableWrap     = "\033[?7l"
	enableWrap      = "\033[?7h"
)

// FullScreen redraws a foreground watch view without appending frames.
func FullScreen(view string) string {
	return "\033[H\033[2J" + view
}

// EraseScrollback removes stored terminal lines without changing the visible
// screen or cursor. It must run after the initial header has been drawn so the
// terminal does not create fresh blank history while establishing its scroll
// region.
func EraseScrollback() string {
	return eraseScrollback
}

// PinnedRegions clears stale terminal content, writes fixed header regions,
// and reserves their rows above the shell's scrolling area. Unlike a
// newline-based view, its output cannot be reflowed into extra rows on resize.
func PinnedRegions(height int, regions ...string) string {
	var output strings.Builder
	output.WriteString(ReleaseHeader())
	output.WriteString(FullScreen(""))
	for _, region := range regions {
		output.WriteString(region)
	}
	output.WriteString(ReserveHeader(height))
	return output.String()
}

// PinnedRegionsPreservingContent supports older shell integrations by redrawing
// only reserved header rows. Callers must keep command output below those rows.
func PinnedRegionsPreservingContent(height int, regions ...string) string {
	var output strings.Builder
	output.WriteString(ReleaseHeader())
	output.WriteString(ClearRows(height))
	for _, region := range regions {
		output.WriteString(region)
	}
	output.WriteString(MaintainHeader(height))
	return output.String()
}

// ReserveHeader fixes the first height rows in place and moves the cursor to
// the first row of the scrolling shell area below them.
func ReserveHeader(height int) string {
	if height < 1 {
		return ""
	}
	top := height + 1
	return fmt.Sprintf("\033[%d;r\033[%d;1H", top, top)
}

// MaintainHeader reapplies the scrolling area while preserving the shell's
// cursor. This keeps the header fixed after terminal resize events.
func MaintainHeader(height int) string {
	if height < 1 {
		return ""
	}
	return saveCursor + fmt.Sprintf("\033[%d;r", height+1) + restoreCursor
}

// ReleaseHeader restores scrolling across the complete terminal without moving
// the shell cursor. Changing the scroll margins otherwise moves it to row one.
func ReleaseHeader() string {
	return saveCursor + "\033[r" + restoreCursor
}

// UnpinHeader removes the fixed rows before restoring full-screen scrolling.
// Otherwise command output can push an old header into terminal scrollback.
func UnpinHeader(height int) string {
	return ClearRows(height) + ReleaseHeader()
}

// ClearRows erases complete terminal rows without changing the shell cursor.
func ClearRows(height int) string {
	if height < 1 {
		return ""
	}
	var output strings.Builder
	output.WriteString(saveCursor)
	for row := 1; row <= height; row++ {
		output.WriteString(fmt.Sprintf("\033[%d;1H", row))
		output.WriteString(eraseLine)
	}
	output.WriteString(restoreCursor)
	return output.String()
}

// Region updates a fixed rectangular terminal area without emitting line
// breaks. Absolute cursor movement prevents live frames from scrolling.
func Region(lines []string, row, column, width int) string {
	var output strings.Builder
	output.WriteString(saveCursor)
	output.WriteString(disableWrap)
	for index, line := range lines {
		output.WriteString(fmt.Sprintf("\033[%d;%dH", row+index, column))
		output.WriteString(fmt.Sprintf("\033[%dX", width))
		output.WriteString(line)
	}
	output.WriteString(enableWrap)
	output.WriteString(restoreCursor)
	return output.String()
}
