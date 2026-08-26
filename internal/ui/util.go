package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"

	"github.com/jinhyo/gitpad/internal/ui/theme"
)

// trunc cuts s to at most w cells, adding an ellipsis when cut.
func trunc(s string, w int) string {
	if w <= 0 {
		return ""
	}
	if ansi.StringWidth(s) <= w {
		return s
	}
	if w == 1 {
		return "…"
	}
	return ansi.Truncate(s, w, "…")
}

// pad right-pads s with spaces to exactly w cells (truncating if longer).
func pad(s string, w int) string {
	if w <= 0 {
		return ""
	}
	sw := ansi.StringWidth(s)
	if sw > w {
		return ansi.Truncate(s, w, "")
	}
	return s + strings.Repeat(" ", w-sw)
}

// padLeft left-pads s to w cells.
func padLeft(s string, w int) string {
	sw := ansi.StringWidth(s)
	if sw >= w {
		return s
	}
	return strings.Repeat(" ", w-sw) + s
}

func width(s string) int { return ansi.StringWidth(s) }

func bgSequence(c lipgloss.TerminalColor) string {
	r := lipgloss.DefaultRenderer()
	var tc termenv.Color
	switch v := c.(type) {
	case lipgloss.AdaptiveColor:
		if r.HasDarkBackground() {
			tc = r.ColorProfile().Color(v.Dark)
		} else {
			tc = r.ColorProfile().Color(v.Light)
		}
	case lipgloss.Color:
		tc = r.ColorProfile().Color(string(v))
	default:
		return ""
	}
	if tc == nil {
		return ""
	}
	seq := tc.Sequence(true)
	if seq == "" {
		return ""
	}
	return "\x1b[" + seq + "m"
}

// highlight paints a background color under an already-styled line, re-applying
// the background after every SGR reset emitted by inner styles.
func highlight(line string, bg lipgloss.TerminalColor) string {
	seq := bgSequence(bg)
	if seq == "" {
		return line
	}
	line = strings.ReplaceAll(line, "\x1b[0m", "\x1b[0m"+seq)
	line = strings.ReplaceAll(line, "\x1b[m", "\x1b[0m"+seq)
	return seq + line + "\x1b[0m"
}

// fmtDate renders a compact, fixed-width (11 cells) timestamp.
func fmtDate(t time.Time) string {
	now := time.Now()
	switch {
	case t.Year() == now.Year() && t.YearDay() == now.YearDay():
		return padLeft("today "+t.Format("15:04"), 11)
	case t.Year() == now.Year():
		return padLeft(fmt.Sprintf("%d/%02d %s", int(t.Month()), t.Day(), t.Format("15:04")), 11)
	default:
		return padLeft(t.Format("2006-01-02"), 11)
	}
}

// fmtRelative renders "3h ago" style durations.
func fmtRelative(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	case d < 30*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	case d < 365*24*time.Hour:
		return fmt.Sprintf("%dmo ago", int(d.Hours()/24/30))
	default:
		return fmt.Sprintf("%dy ago", int(d.Hours()/24/365))
	}
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ensureVisible scrolls so that cursor is inside [scroll, scroll+height).
func ensureVisible(cursor int, scroll *int, height int) {
	if height <= 0 {
		return
	}
	if cursor < *scroll {
		*scroll = cursor
	}
	if cursor >= *scroll+height {
		*scroll = cursor - height + 1
	}
	if *scroll < 0 {
		*scroll = 0
	}
}

// keyHints renders "key label" pairs for the status bar.
func keyHints(pairs ...string) string {
	var parts []string
	for i := 0; i+1 < len(pairs); i += 2 {
		parts = append(parts, theme.KeyHint.Render(pairs[i])+" "+theme.KeyLabel.Render(pairs[i+1]))
	}
	return strings.Join(parts, theme.DimSt.Render("  "))
}

func plural(n int, one, many string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, one)
	}
	return fmt.Sprintf("%d %s", n, many)
}
