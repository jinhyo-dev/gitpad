package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/jinhyo-dev/gitpad/internal/ci"
	"github.com/jinhyo-dev/gitpad/internal/git"
	"github.com/jinhyo-dev/gitpad/internal/graph"
	"github.com/jinhyo-dev/gitpad/internal/ui/theme"
)

// selectionBg picks the highlight color for a panel's cursor row.
func (m *Model) selectionBg(p Panel) lipgloss.TerminalColor {
	if m.focus == p {
		return theme.Selection
	}
	return theme.SelDim
}

// ---- branches ----------------------------------------------------------

func (m *Model) renderBranches(w, h int) []string {
	ensureVisible(m.bcur, &m.bscroll, h)
	var out []string
	end := minInt(len(m.bnodes), m.bscroll+h)
	for i := m.bscroll; i < end; i++ {
		n := m.bnodes[i]
		line := m.renderBranchRow(n, w)
		if i == m.bcur {
			line = highlight(line, m.selectionBg(PanelBranches))
		}
		out = append(out, line)
	}
	return out
}

func (m *Model) renderBranchRow(n bnode, w int) string {
	indent := strings.Repeat("  ", n.depth)
	switch n.kind {
	case bHead:
		icon := lipgloss.NewStyle().Foreground(theme.Accent).Render("◉ ")
		label := theme.Bold.Render(trunc(n.label, w-4))
		extra := ""
		if m.info.Ahead > 0 || m.info.Behind > 0 {
			extra = m.aheadBehind(m.info.Ahead, m.info.Behind)
		}
		return joinRow(" "+icon+label, extra, w)
	case bSection:
		arrow := "▾"
		if !m.bexp[n.key] {
			arrow = "▸"
		}
		title := theme.FrameTitle.Render(n.label)
		count := theme.FrameCount.Render(fmt.Sprintf("%d", n.count))
		return joinRow(" "+theme.MutedSt.Render(arrow)+" "+title, count+" ", w)
	case bFolder:
		arrow := "▾"
		if !m.bexp[n.key] {
			arrow = "▸"
		}
		return joinRow(" "+indent+theme.MutedSt.Render(arrow)+" "+theme.Base.Render(trunc(n.label, w-4-len(indent))), theme.FrameCount.Render(fmt.Sprintf("%d ", n.count)), w)
	default:
		b := n.branch
		icon := "  "
		st := theme.Base
		switch b.Kind {
		case git.RefTag:
			st = lipgloss.NewStyle().Foreground(theme.Yellow)
		case git.RefRemote:
			st = theme.Base
		}
		if b.IsHead {
			icon = lipgloss.NewStyle().Foreground(theme.Accent).Render("● ")
			st = theme.AccentB
		}
		extra := ""
		if b.Kind == git.RefLocal {
			if b.Gone {
				extra = theme.DimSt.Render("gone ")
			} else if b.Ahead > 0 || b.Behind > 0 {
				extra = m.aheadBehind(b.Ahead, b.Behind)
			}
		}
		return joinRow(" "+indent+icon+st.Render(trunc(n.label, w-4-len(indent))), extra, w)
	}
}

func (m *Model) aheadBehind(a, b int) string {
	var parts []string
	if a > 0 {
		parts = append(parts, lipgloss.NewStyle().Foreground(theme.Green).Render(fmt.Sprintf("↑%d", a)))
	}
	if b > 0 {
		parts = append(parts, lipgloss.NewStyle().Foreground(theme.Orange).Render(fmt.Sprintf("↓%d", b)))
	}
	return strings.Join(parts, " ") + " "
}

// joinRow places right at the end of the row, truncating left as needed.
func joinRow(left, right string, w int) string {
	rw := width(right)
	if rw >= w {
		return pad(left, w)
	}
	lw := w - rw
	if width(left) > lw {
		left = ansi.Truncate(left, lw-1, "…")
	}
	return pad(left, lw) + right
}

// ---- log ---------------------------------------------------------------

const (
	colAuthor = 14
	colDate   = 11
)

func (m *Model) renderLog(w, h int) []string {
	ensureVisible(m.lcur, &m.lscroll, h)
	var out []string
	n := m.logLen()
	if n == 0 && !m.loaded {
		return []string{"", "  " + m.spin.View() + theme.MutedSt.Render(" loading repository…")}
	}
	if n == 0 {
		msg := "No commits match the current filter"
		if !m.hasFilter() {
			msg = "No commits yet"
		}
		out = append(out, "", theme.MutedSt.Render(pad("  "+msg, w)))
		return out
	}
	end := minInt(n, m.lscroll+h)
	for i := m.lscroll; i < end; i++ {
		var line string
		if m.isLocalRow(i) {
			line = m.renderLocalRow(w)
		} else {
			line = m.renderCommitRow(m.commitAt(i), w)
		}
		if i == m.lcur {
			line = highlight(line, m.selectionBg(PanelLog))
		}
		out = append(out, line)
	}
	return out
}

func (m *Model) narrowLog(w int) bool { return w < 70 }

// maxGraphCells caps the graph column so deep histories don't squeeze the
// subject column; rows whose node falls beyond the cap are compressed.
const maxGraphCells = 16

func (m *Model) renderGraphCells(row int, w int) string {
	if row < 0 || row >= len(m.rows) {
		return strings.Repeat(" ", w)
	}
	var sb strings.Builder
	r := m.rows[row]
	cells := r.Cells
	isHead := m.commits[row].Hash == m.info.HeadHash
	draw := func(c graph.Cell) {
		if c.Rune == ' ' {
			sb.WriteRune(' ')
			return
		}
		st := theme.Lane(c.Color)
		g := c.Rune
		if g == '●' && isHead {
			g = '◉'
			st = st.Bold(true)
		}
		sb.WriteString(st.Render(string(g)))
	}
	cw := 0
	if r.NodeCol >= w {
		// Compress: leading lanes, an ellipsis, then the node itself.
		for _, c := range cells[:w-2] {
			draw(c)
			cw++
		}
		sb.WriteString(theme.DimSt.Render("┄"))
		draw(cells[r.NodeCol])
		cw += 2
	} else {
		for _, c := range cells {
			if cw >= w {
				break
			}
			draw(c)
			cw++
		}
	}
	for ; cw < w; cw++ {
		sb.WriteByte(' ')
	}
	return sb.String()
}

func (m *Model) renderCommitRow(c *git.Commit, w int) string {
	if c == nil {
		return pad("", w)
	}
	idx := m.hashIdx[c.Hash]
	isHead := c.Hash == m.info.HeadHash
	var sb strings.Builder
	sb.WriteByte(' ')
	rest := w - 1
	if !m.narrowLog(w) {
		author := theme.MutedSt.Render(pad(trunc(c.Author, colAuthor), colAuthor))
		if isHead {
			author = theme.Base.Render(pad(trunc(c.Author, colAuthor), colAuthor))
		}
		sb.WriteString(author + " ")
		sb.WriteString(theme.DimSt.Render(fmtDate(c.When)) + " ")
		rest -= colAuthor + 1 + colDate + 1
		if m.ci != nil {
			sb.WriteString(m.ciGlyph(c.Hash) + " ")
			rest -= 2
		}
	}
	sb.WriteString(m.renderGraphCells(idx, m.graphW) + " ")
	rest -= m.graphW + 1

	chips := renderChips(c.Refs, m.info.Detached)
	cw := width(chips)
	if cw > rest*45/100 {
		chips = ansi.Truncate(chips, rest*45/100, "…")
		cw = width(chips)
	}
	subjW := rest
	if cw > 0 {
		subjW = rest - cw - 1
	}
	subjSt := theme.Base
	switch {
	case isHead:
		subjSt = theme.Bold
	case c.IsMerge():
		subjSt = theme.MutedSt
	}
	sb.WriteString(pad(subjSt.Render(trunc(c.Subject, subjW)), subjW))
	if cw > 0 {
		sb.WriteString(" " + chips)
	}
	return pad(sb.String(), w)
}

func (m *Model) renderLocalRow(w int) string {
	var sb strings.Builder
	sb.WriteByte(' ')
	rest := w - 1
	if !m.narrowLog(w) {
		sb.WriteString(pad("", colAuthor+1+colDate+1))
		rest -= colAuthor + 1 + colDate + 1
		if m.ci != nil {
			sb.WriteString("  ")
			rest -= 2
		}
	}
	col := 0
	if i, ok := m.hashIdx[m.info.HeadHash]; ok && i < len(m.rows) {
		col = m.rows[i].NodeCol
	}
	g := strings.Repeat(" ", col) + lipgloss.NewStyle().Foreground(theme.Yellow).Render("◌")
	sb.WriteString(pad(g, m.graphW) + " ")
	rest -= m.graphW + 1
	label := lipgloss.NewStyle().Foreground(theme.Yellow).Italic(true).Render("Uncommitted changes")
	count := theme.MutedSt.Render(" · " + plural(len(m.status), "file", "files"))
	sb.WriteString(pad(trunc(label+count, rest), rest))
	return pad(sb.String(), w)
}

// ciGlyph renders the build status marker for a commit.
func (m *Model) ciGlyph(hash string) string {
	r, ok := m.ciResults[hash]
	if !ok {
		if m.ciPending[hash] {
			return theme.DimSt.Render("·")
		}
		return " "
	}
	return ciStateGlyph(r.State)
}

func ciStateGlyph(s ci.State) string {
	switch s {
	case ci.StateSuccess:
		return lipgloss.NewStyle().Foreground(theme.Green).Render("✓")
	case ci.StateFailure:
		return lipgloss.NewStyle().Foreground(theme.Red).Render("✗")
	case ci.StatePending:
		return lipgloss.NewStyle().Foreground(theme.Teal).Render("◌")
	}
	return " "
}

func renderChips(refs []git.Ref, detached bool) string {
	if len(refs) == 0 {
		return ""
	}
	var parts []string
	for _, r := range refs {
		switch r.Kind {
		case git.RefHead:
			if detached {
				parts = append(parts, theme.ChipHead.Render("HEAD"))
			} else {
				parts = append(parts, theme.ChipHead.Render("HEAD"))
			}
		case git.RefLocal:
			parts = append(parts, theme.ChipLocal.Render(r.Name))
		case git.RefRemote:
			parts = append(parts, theme.ChipRemote.Render(r.Name))
		case git.RefTag:
			parts = append(parts, theme.ChipTag.Render("⌂ "+r.Name))
		}
	}
	return strings.Join(parts, " ")
}

// ---- changes -----------------------------------------------------------

func (m *Model) renderFiles(w, h int) []string {
	ensureVisible(m.fcur, &m.fscroll, h)
	var out []string
	if len(m.fnodes) == 0 {
		msg := "No changes"
		if m.filesFor == "" {
			msg = "Select a commit"
		}
		out = append(out, "", theme.MutedSt.Render(pad("  "+msg, w)))
		return out
	}
	end := minInt(len(m.fnodes), m.fscroll+h)
	for i := m.fscroll; i < end; i++ {
		n := m.fnodes[i]
		line := m.renderFileRow(n, w)
		if i == m.fcur {
			line = highlight(line, m.selectionBg(PanelChanges))
		}
		out = append(out, line)
	}
	return out
}

func (m *Model) checkbox(on bool) string {
	if on {
		return lipgloss.NewStyle().Foreground(theme.Accent).Bold(true).Render("[x]")
	}
	return theme.DimSt.Render("[ ]")
}

func (m *Model) renderFileRow(n fnode, w int) string {
	indent := strings.Repeat("  ", n.depth)
	local := m.filesFor == "local"
	if n.isDir {
		arrow := "▾"
		if m.fcollapsed[n.key] {
			arrow = "▸"
		}
		label := theme.Base.Render(trunc(n.label, w-6-len(indent)))
		if n.isGroup {
			label = theme.FrameTitle.Render(trunc(n.label, w-6))
		}
		left := " " + indent + theme.MutedSt.Render(arrow) + " " + label
		count := plural(n.count, "file", "files")
		if local {
			sel := 0
			for _, p := range m.nodePaths(&n) {
				if m.selected[p] {
					sel++
				}
			}
			count = fmt.Sprintf("%d/%d", sel, n.count)
		}
		return joinRow(left, theme.FrameCount.Render(count+" "), w)
	}
	f := n.file
	st := theme.Status(f.Status)
	badge := st.Bold(true).Render(string(f.Status))
	if f.Status == '?' {
		badge = st.Render("?")
	}
	if local {
		badge = m.checkbox(m.selected[f.Path]) + " " + badge
	}
	label := st.Render(trunc(n.label, w-10-len(indent)))
	extra := ""
	if f.Conflict {
		extra = lipgloss.NewStyle().Foreground(theme.Red).Render("conflict ")
	} else if f.OldPath != "" {
		extra = theme.DimSt.Render("← " + trunc(f.OldPath, 18) + " ")
	}
	return joinRow(" "+indent+badge+" "+label, extra, w)
}

func (m *Model) renderDetails(w, h int) []string {
	var lines []string
	if m.filesFor == "local" {
		staged, unstaged, untracked := 0, 0, 0
		for _, f := range m.status {
			switch {
			case f.Status == '?':
				untracked++
			case f.Staged:
				staged++
			default:
				unstaged++
			}
		}
		sel := len(m.selectedFiles())
		lines = append(lines, theme.Bold.Render("Uncommitted changes"), "")
		lines = append(lines, theme.MutedSt.Render(fmt.Sprintf("%d changed · %d untracked · %d staged", unstaged+staged, untracked, staged)))
		lines = append(lines, lipgloss.NewStyle().Foreground(theme.Accent).Render(fmt.Sprintf("%d of %d files checked for commit", sel, len(m.status))))
		lines = append(lines, "")
		lines = append(lines, theme.DimSt.Render("space  check file     a  check all"))
		lines = append(lines, theme.DimSt.Render("c      commit…        P  push…"))
		lines = append(lines, theme.DimSt.Render("enter  diff           m  more actions"))
	} else if d := m.details; d != nil {
		msg := strings.SplitN(d.Message, "\n", 2)
		for _, l := range strings.Split(ansi.Wordwrap(msg[0], w, ""), "\n") {
			lines = append(lines, theme.Bold.Render(l))
		}
		if len(msg) > 1 && strings.TrimSpace(msg[1]) != "" {
			lines = append(lines, "")
			for _, l := range strings.Split(ansi.Wordwrap(strings.TrimSpace(msg[1]), w, ""), "\n") {
				lines = append(lines, theme.MutedSt.Render(l))
			}
		}
		lines = append(lines, "")
		hash := lipgloss.NewStyle().Foreground(theme.Accent).Render(d.Hash[:minInt(12, len(d.Hash))])
		lines = append(lines, hash+" "+theme.Base.Render(d.Author)+" "+theme.DimSt.Render("<"+d.Email+">"))
		lines = append(lines, theme.MutedSt.Render(d.AuthorAt.Format("2006-01-02 15:04")+" · "+fmtRelative(d.AuthorAt)))
		if len(d.Parents) > 1 {
			var ps []string
			for _, p := range d.Parents {
				ps = append(ps, p[:minInt(8, len(p))])
			}
			lines = append(lines, theme.DimSt.Render("merge of "+strings.Join(ps, " + ")))
		}
		if len(d.Refs) > 0 {
			lines = append(lines, "")
			lines = append(lines, renderChips(d.Refs, m.info.Detached))
		}
		if r, ok := m.ciResults[d.Hash]; ok && len(r.Checks) > 0 {
			lines = append(lines, "")
			summary := map[ci.State]string{ci.StateSuccess: "all checks passed", ci.StateFailure: "checks failed", ci.StatePending: "checks running"}[r.State]
			lines = append(lines, theme.FrameTitle.Render("Checks")+"  "+ciStateGlyph(r.State)+" "+theme.MutedSt.Render(summary))
			const maxChecks = 8
			for i, c := range r.Checks {
				if i >= maxChecks {
					lines = append(lines, theme.DimSt.Render(fmt.Sprintf("  … +%d more", len(r.Checks)-maxChecks)))
					break
				}
				line := "  " + ciStateGlyph(c.State) + " " + theme.Base.Render(trunc(c.Name, w-14))
				if c.Duration != "" {
					line += theme.DimSt.Render("  " + c.Duration)
				}
				lines = append(lines, line)
			}
		}
	} else {
		lines = append(lines, theme.MutedSt.Render("Select a commit"))
	}
	out := make([]string, 0, len(lines))
	for _, l := range lines {
		for _, wrapped := range strings.Split(ansi.Hardwrap(l, w, true), "\n") {
			out = append(out, pad(" "+wrapped, w))
		}
	}
	if m.dscroll > maxInt(0, len(out)-h) {
		m.dscroll = maxInt(0, len(out)-h)
	}
	if m.dscroll > 0 && m.dscroll < len(out) {
		out = out[m.dscroll:]
	}
	return out
}
