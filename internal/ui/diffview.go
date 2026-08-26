package ui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/jinhyo-dev/gitpad/internal/git"
	"github.com/jinhyo-dev/gitpad/internal/ui/theme"
)

type diffLine struct {
	kind  byte // '+', '-', ' ', '@', 'h' (header)
	oldNo int
	newNo int
	text  string
}

type diffState struct {
	path    string
	commit  *git.Commit
	file    git.FileChange
	lines   []diffLine
	scroll  int
	loading bool
	err     error
	stats   [2]int
}

func parseDiff(text string) ([]diffLine, [2]int) {
	var lines []diffLine
	var stats [2]int
	oldNo, newNo := 0, 0
	inHunk := false
	for _, raw := range strings.Split(strings.TrimRight(text, "\n"), "\n") {
		switch {
		case strings.HasPrefix(raw, "@@"):
			inHunk = true
			oldNo, newNo = parseHunk(raw)
			lines = append(lines, diffLine{kind: '@', text: raw})
		case !inHunk:
			lines = append(lines, diffLine{kind: 'h', text: raw})
		case strings.HasPrefix(raw, "+"):
			lines = append(lines, diffLine{kind: '+', newNo: newNo, text: raw[1:]})
			newNo++
			stats[0]++
		case strings.HasPrefix(raw, "-"):
			lines = append(lines, diffLine{kind: '-', oldNo: oldNo, text: raw[1:]})
			oldNo++
			stats[1]++
		case strings.HasPrefix(raw, "\\"):
			lines = append(lines, diffLine{kind: 'h', text: raw})
		default:
			t := raw
			if strings.HasPrefix(t, " ") {
				t = t[1:]
			}
			lines = append(lines, diffLine{kind: ' ', oldNo: oldNo, newNo: newNo, text: t})
			oldNo++
			newNo++
		}
	}
	return lines, stats
}

func parseHunk(s string) (int, int) {
	// @@ -a,b +c,d @@
	f := strings.Fields(s)
	old, nw := 0, 0
	for _, p := range f[1:] {
		if strings.HasPrefix(p, "-") {
			old, _ = strconv.Atoi(strings.SplitN(p[1:], ",", 2)[0])
		} else if strings.HasPrefix(p, "+") {
			nw, _ = strconv.Atoi(strings.SplitN(p[1:], ",", 2)[0])
		} else if p == "@@" {
			break
		}
	}
	return old, nw
}

func (m *Model) renderDiff(w, h int) []string {
	d := m.diff
	var out []string
	if d == nil {
		return out
	}
	if d.err != nil {
		out = append(out, theme.MenuDanger.Render(trunc("  "+d.err.Error(), w)))
		return out
	}
	if d.loading {
		out = append(out, theme.MutedSt.Render("  loading diff…"))
		return out
	}
	if len(d.lines) == 0 {
		out = append(out, theme.MutedSt.Render("  (no textual changes — binary or mode-only)"))
		return out
	}
	gutter := 5
	end := minInt(len(d.lines), d.scroll+h)
	for _, l := range d.lines[d.scroll:end] {
		text := strings.ReplaceAll(l.text, "\t", "    ")
		var line string
		switch l.kind {
		case '@':
			line = pad(theme.DiffHunk.Render(trunc(text, w)), w)
		case 'h':
			line = pad(theme.DiffMeta.Render(trunc(text, w)), w)
		case '+':
			g := theme.DiffGutter.Render(padLeft("", gutter) + padLeft(strconv.Itoa(l.newNo), gutter))
			body := theme.DiffAdd.Render(" +" + trunc(text, w-gutter*2-2))
			line = highlight(pad(g+body, w), theme.DiffAddBg)
		case '-':
			g := theme.DiffGutter.Render(padLeft(strconv.Itoa(l.oldNo), gutter) + padLeft("", gutter))
			body := theme.DiffDel.Render(" -" + trunc(text, w-gutter*2-2))
			line = highlight(pad(g+body, w), theme.DiffDelBg)
		default:
			g := theme.DiffGutter.Render(padLeft(strconv.Itoa(l.oldNo), gutter) + padLeft(strconv.Itoa(l.newNo), gutter))
			line = pad(g+theme.Base.Render("  "+trunc(text, w-gutter*2-2)), w)
		}
		out = append(out, line)
	}
	return out
}

func (m *Model) diffTitle() string {
	d := m.diff
	if d == nil {
		return "Diff"
	}
	stats := lipgloss.NewStyle().Foreground(theme.Green).Render(fmt.Sprintf("+%d", d.stats[0])) + " " +
		lipgloss.NewStyle().Foreground(theme.Red).Render(fmt.Sprintf("-%d", d.stats[1]))
	return "Diff · " + d.path + "  " + stats
}

// diffMaxScroll bounds scrolling.
func (m *Model) diffMaxScroll() int {
	if m.diff == nil {
		return 0
	}
	return maxInt(0, len(m.diff.lines)-m.diffRect.h)
}
