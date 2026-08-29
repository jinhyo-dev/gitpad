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
	// blocks are runs of consecutive added/removed lines; ↑/↓ jump between
	// them and the current one is marked in the gutter.
	blocks [][2]int // [start, end) line indices
	cur    int
	// hunks are the @@ sections (staging granularity in the commit workspace)
	hunks   [][2]int // [start, end) line indices, start is the @@ line
	curHunk int
}

// hunkRanges finds the @@ sections of a parsed diff.
func hunkRanges(lines []diffLine) [][2]int {
	var out [][2]int
	for i, l := range lines {
		if l.kind == '@' {
			if n := len(out); n > 0 {
				out[n-1][1] = i
			}
			out = append(out, [2]int{i, len(lines)})
		}
	}
	return out
}

// hunkAt returns the hunk index containing line i (-1 when none).
func (d *diffState) hunkAt(i int) int {
	for h, r := range d.hunks {
		if i >= r[0] && i < r[1] {
			return h
		}
	}
	return -1
}

// diffJumpHunk moves the current hunk (commit workspace navigation).
func (m *Model) diffJumpHunk(dir int) {
	d := m.diff
	if d == nil || len(d.hunks) == 0 {
		return
	}
	d.curHunk = clamp(d.curHunk+dir, 0, len(d.hunks)-1)
	d.scroll = clamp(d.hunks[d.curHunk][0], 0, m.diffMaxScroll())
}

// changeBlocks finds runs of +/- lines.
func changeBlocks(lines []diffLine) [][2]int {
	var blocks [][2]int
	start := -1
	for i, l := range lines {
		changed := l.kind == '+' || l.kind == '-'
		switch {
		case changed && start < 0:
			start = i
		case !changed && start >= 0:
			blocks = append(blocks, [2]int{start, i})
			start = -1
		}
	}
	if start >= 0 {
		blocks = append(blocks, [2]int{start, len(lines)})
	}
	return blocks
}

// diffJump moves to the next (+1) or previous (-1) change block and scrolls
// so it sits two lines below the top of the viewport.
func (m *Model) diffJump(dir int) {
	d := m.diff
	if d == nil || len(d.blocks) == 0 {
		return
	}
	d.cur = clamp(d.cur+dir, 0, len(d.blocks)-1)
	d.scroll = clamp(d.blocks[d.cur][0]-2, 0, m.diffMaxScroll())
}

// inCurrentBlock reports whether line i belongs to the highlighted block.
func (d *diffState) inCurrentBlock(i int) bool {
	if len(d.blocks) == 0 || d.cur >= len(d.blocks) {
		return false
	}
	b := d.blocks[d.cur]
	return i >= b[0] && i < b[1]
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
	w-- // one column for the current-block marker
	staging := m.commitOpen && m.filesFor == "local"
	end := minInt(len(d.lines), d.scroll+h)
	for i := d.scroll; i < end; i++ {
		l := d.lines[i]
		text := strings.ReplaceAll(l.text, "\t", "    ")
		var line string
		switch l.kind {
		case '@':
			if staging {
				h := d.hunkAt(i)
				box := m.checkbox(m.hunkSelected(d.path, h))
				line = pad(box+" "+theme.DiffHunk.Render(trunc(text, w-4)), w)
			} else {
				line = pad(theme.DiffHunk.Render(trunc(text, w)), w)
			}
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
		marker := " "
		if staging {
			if len(d.hunks) > 0 && d.hunkAt(i) == d.curHunk {
				marker = theme.AccentB.Render("▎")
			}
		} else if d.inCurrentBlock(i) {
			marker = theme.AccentB.Render("▎")
		}
		out = append(out, marker+line)
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
	pos := ""
	switch {
	case m.commitOpen && m.filesFor == "local" && len(d.hunks) > 0:
		sel := 0
		for h := range d.hunks {
			if m.hunkSelected(d.path, h) {
				sel++
			}
		}
		pos = theme.DimSt.Render(fmt.Sprintf("  hunk %d/%d · %d checked", d.curHunk+1, len(d.hunks), sel))
	case len(d.blocks) > 0:
		pos = theme.DimSt.Render(fmt.Sprintf("  change %d/%d", d.cur+1, len(d.blocks)))
	}
	return "Diff · " + d.path + "  " + stats + pos
}

// diffMaxScroll bounds scrolling.
func (m *Model) diffMaxScroll() int {
	if m.diff == nil {
		return 0
	}
	return maxInt(0, len(m.diff.lines)-m.diffRect.h)
}
