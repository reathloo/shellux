package render

import (
	"fmt"
	"strings"
	"time"
)

const focusClockWidth, focusClockHeight = 51, 26
const focusGlyphWidth, focusGlyphHeight = 10, 8

// Glyphs from the clock font supplied by the user. Every digit occupies ten
// columns, while the colon occupies eight.
var focusDigits = [10][focusGlyphHeight]string{
	{` ######\  `, `### __##\ `, `####\ ## |`, `##\##\## |`, `## \#### |`, `## |\### |`, `\######  /`, ` \______/ `},
	{`   ##\    `, ` #### |   `, ` \_## |   `, `   ## |   `, `   ## |   `, `   ## |   `, ` ######\  `, ` \______| `},
	{` ######\  `, `##  __##\ `, `\__/  ## |`, ` ######  |`, `##  ____/ `, `## |      `, `########\ `, `\________|`},
	{` ######\  `, `## ___##\ `, `\_/   ## |`, `  ##### / `, `  \___##\ `, `##\   ## |`, `\######  |`, ` \______/ `},
	{`##\   ##\ `, `## |  ## |`, `## |  ## |`, `######## |`, `\_____## |`, `      ## |`, `      ## |`, `      \__|`},
	{`#######\  `, `##  ____| `, `## |      `, `#######\  `, `\_____##\ `, `##\   ## |`, `\######  |`, ` \______/ `},
	{` ######\  `, `##  __##\ `, `## /  \__|`, `#######\  `, `##  __##\ `, `## /  ## |`, ` ######  |`, ` \______/ `},
	{`########\ `, `\____##  |`, `    ##  / `, `   ##  /  `, `  ##  /   `, ` ##  /    `, `##  /     `, `\__/      `},
	{` ######\  `, `##  __##\ `, `## /  ## |`, ` ######  |`, `##  __##< `, `## /  ## |`, `\######  |`, ` \______/ `},
	{` ######\  `, `##  __##\ `, `## /  ## |`, `\####### |`, ` \____## |`, `##\   ## |`, `\######  |`, ` \______/ `},
}

var focusColon = [focusGlyphHeight]string{"        ", "        ", `  ##\   `, `  \__|  `, "        ", `  ##\   `, `  \__|  `, "        "}

type clockCell struct {
	letter rune
	shade  byte
}

var clockShades = [...]string{
	"", "\x1b[38;2;91;102;115m", "\x1b[38;2;124;136;149m",
	"\x1b[38;2;174;186;198m", "\x1b[38;2;210;220;230m",
}

func focusClockText(remaining time.Duration) string {
	if remaining < 0 {
		remaining = 0
	}
	if remaining < time.Hour {
		seconds := min(int((remaining+time.Second-1)/time.Second), 59*60+59)
		return fmt.Sprintf("%02d:%02d", seconds/60, seconds%60)
	}
	minutes := int((remaining + time.Minute - 1) / time.Minute)
	return fmt.Sprintf("%02d:%02d", minutes/60, minutes%60)
}

// FocusClock uses the reference's shaded ASCII digits on the standard 51x26
// canvas. Below one hour it shows minutes and seconds, then hours and minutes.
func FocusClock(remaining time.Duration) string {
	clock := focusClockText(remaining)
	var canvas [focusClockHeight][focusClockWidth]clockCell
	place := func(x int, glyph [focusGlyphHeight]string) {
		for row, line := range glyph {
			for col, character := range line {
				if col >= focusGlyphWidth || character == ' ' {
					continue
				}
				shade := byte(2)
				if character == '#' {
					shade = 3
					if row == 0 || row == 3 || row == 6 {
						shade = 4
					}
				} else if character == '_' {
					shade = 1
				}
				canvas[9+row][x+col] = clockCell{character, shade}
			}
		}
	}
	// Two empty columns separate the minute digits for easier reading.
	for i, x := range [4]int{0, 12, 29, 39} {
		digitIndex := i
		if i >= 2 {
			digitIndex++ // Skip the colon in 00:00.
		}
		place(x, focusDigits[clock[digitIndex]-'0'])
	}
	place(21, focusColon)

	lines := make([]string, focusClockHeight)
	for y, row := range canvas {
		var line strings.Builder
		currentShade := byte(0)
		for _, cell := range row {
			if cell.shade != currentShade {
				if cell.shade == 0 {
					line.WriteString("\x1b[0m")
				} else {
					line.WriteString(clockShades[cell.shade])
				}
				currentShade = cell.shade
			}
			if cell.letter == 0 {
				line.WriteByte(' ')
			} else {
				line.WriteRune(cell.letter)
			}
		}
		if currentShade != 0 {
			line.WriteString("\x1b[0m")
		}
		lines[y] = line.String()
	}
	return strings.Join(lines, "\n")
}
