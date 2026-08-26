// Package overlay composes a foreground block on top of a background block,
// preserving ANSI styling on both sides of the cut.
package overlay

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// Compose places fg over bg with its top-left corner at column x, row y.
func Compose(bg, fg string, x, y int) string {
	bgLines := strings.Split(bg, "\n")
	fgLines := strings.Split(fg, "\n")
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	for i, fl := range fgLines {
		row := y + i
		if row >= len(bgLines) {
			break
		}
		fw := ansi.StringWidth(fl)
		bl := bgLines[row]
		bw := ansi.StringWidth(bl)
		if bw > 0 && x+fw > bw {
			// Clip the overlay to the background width.
			fl = ansi.Truncate(fl, maxInt(0, bw-x), "")
			fw = ansi.StringWidth(fl)
		}
		left := ansi.Truncate(bl, x, "")
		if lw := ansi.StringWidth(left); lw < x {
			// A wide glyph straddled the cut; fill the gap.
			left += strings.Repeat(" ", x-lw)
		}
		right := ""
		if want := bw - (x + fw); want > 0 {
			right = ansi.TruncateLeft(bl, x+fw, "")
			// A wide glyph straddling the right edge is kept whole by
			// TruncateLeft, making the line too wide: cut one cell further.
			if ansi.StringWidth(right) > want {
				right = ansi.TruncateLeft(bl, x+fw+1, "")
			}
			if ansi.StringWidth(right) > want {
				right = "" // give up on the fragment rather than break layout
			}
			if rw := ansi.StringWidth(right); rw < want {
				right = strings.Repeat(" ", want-rw) + right
			}
		}
		bgLines[row] = left + "\x1b[0m" + fl + "\x1b[0m" + right
	}
	return strings.Join(bgLines, "\n")
}

// Center places fg in the middle of a width×height background.
func Center(bg, fg string, width, height int) string {
	fw, fh := blockSize(fg)
	x := (width - fw) / 2
	y := (height - fh) / 2
	return Compose(bg, fg, x, y)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func blockSize(s string) (w, h int) {
	for _, l := range strings.Split(s, "\n") {
		if lw := ansi.StringWidth(l); lw > w {
			w = lw
		}
		h++
	}
	return
}
