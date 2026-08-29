package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/jinhyo-dev/gitpad/internal/ui/overlay"
	"github.com/jinhyo-dev/gitpad/internal/ui/theme"
)

const (
	rowHeader = 0
	rowFilter = 1
	rowPanels = 2
)

// layout computes panel rectangles for the current size.
func (m *Model) layout() {
	w, h := m.width, m.height
	panelH := maxInt(h-3, 5)
	leftW := clamp(w*22/100, 24, 42)
	rightW := clamp(w*30/100, 30, 64)
	if w < 110 {
		leftW = clamp(w*22/100, 20, 26)
		rightW = clamp(w*30/100, 24, 34)
	}
	if m.commitOpen {
		// Commit mode: the checklist needs little width, the diff needs a lot.
		centerW := clamp((w-leftW)*40/100, 48, 90)
		rightW = w - leftW - centerW
	}
	centerW := maxInt(w-leftW-rightW, 20)
	filesH := maxInt(panelH*55/100, 6)
	detailsH := panelH - filesH

	m.rects[PanelBranches] = rect{x: 1, y: rowPanels + 1, w: leftW - 2, h: panelH - 2}
	m.rects[PanelLog] = rect{x: leftW + 1, y: rowPanels + 1, w: centerW - 2, h: panelH - 2}
	m.rects[PanelChanges] = rect{x: leftW + centerW + 1, y: rowPanels + 1, w: rightW - 2, h: filesH - 2}
	m.detailsRect = rect{x: leftW + centerW + 1, y: rowPanels + filesH + 1, w: rightW - 2, h: detailsH - 2}
	if m.commitOpen {
		m.diffRect = rect{x: leftW + centerW + 1, y: rowPanels + 1, w: rightW - 2, h: panelH - 2}
	} else {
		m.diffRect = m.rects[PanelLog]
	}
	if m.commit != nil {
		_, msgRect, _ := m.commitLayout()
		m.commit.msg.SetWidth(msgRect.w)
		m.commit.msg.SetHeight(msgRect.h)
	}
}

// View renders the whole screen.
func (m Model) View() string {
	if m.fatal != nil {
		box := theme.DialogBoxDanger.Render(theme.DialogTitle.Render("gitpad could not open a repository") + "\n\n" + theme.DialogBody.Render(m.fatal.Error()) + "\n\n" + theme.DimSt.Render("run gitpad inside a git repository, or: gitpad <path>"))
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
	}
	if !m.ready {
		return ""
	}
	mm := &m
	header := mm.renderHeader()
	filter := mm.renderFilterBar()
	panels := mm.renderPanels()
	status := mm.renderStatusBar()
	base := strings.Join([]string{header, filter, panels, status}, "\n")

	if mm.help {
		base = overlay.Center(base, mm.renderHelp(), m.width, m.height)
	}
	if mm.menu != nil {
		var chain []*menu
		for mn := mm.menu; mn != nil; mn = mn.parent {
			chain = append([]*menu{mn}, chain...)
		}
		for _, mn := range chain {
			base = overlay.Compose(base, mn.render(), mn.x, mn.y)
		}
	}
	if mm.push != nil {
		x, y, _, _ := mm.pushBox()
		base = overlay.Compose(base, mm.renderPush(), x, y)
	}
	if mm.dialog != nil {
		base = overlay.Center(base, mm.dialog.render(), m.width, m.height)
	}
	if mm.actions > 0 {
		base = overlay.Center(base, mm.renderProgress(), m.width, m.height)
	}
	if mm.toast != nil {
		pill := mm.renderToast()
		base = overlay.Compose(base, pill, (m.width-width(pill))/2, rowPanels)
	}
	return base
}

// renderProgress is the centered "working" box shown during git operations.
func (m *Model) renderProgress() string {
	label := m.actionLabel
	if label == "" {
		label = "Working"
	}
	body := m.spin.View() + " " + theme.Bold.Render(label+"…")
	return theme.ProgressBox.Render(pad(body, maxInt(28, width(body))))
}

// renderToast is the top-center notification banner.
func (m *Model) renderToast() string {
	t := m.toast
	icon, st := "ℹ", theme.ToastInfo
	switch t.kind {
	case 1:
		icon, st = "✓", theme.ToastOK
	case 2:
		icon, st = "✗", theme.ToastErr
	}
	return st.Render(icon + "  " + trunc(t.text, maxInt(20, m.width-12)))
}

func (m *Model) renderHeader() string {
	left := logoMark() + " " + theme.Logo.Render("gitpad") + " " + theme.Bold.Render(m.info.Name) + theme.DimSt.Render("  on  ")
	branch := lipgloss.NewStyle().Foreground(theme.Accent).Bold(true).Render(m.info.Head)
	if m.info.Detached {
		branch = lipgloss.NewStyle().Foreground(theme.Orange).Bold(true).Render("detached @ " + m.info.Head)
	}
	left += branch
	if m.info.Upstream != "" {
		left += theme.DimSt.Render(" → " + m.info.Upstream)
	}
	if m.info.Ahead > 0 || m.info.Behind > 0 {
		left += " " + strings.TrimSpace(m.aheadBehind(m.info.Ahead, m.info.Behind))
	}
	if m.info.State != "" {
		left += "  " + theme.StateBadge.Render(strings.ToUpper(m.info.State))
	}
	tabLog, tabCon := theme.TabInactive, theme.TabInactive
	if m.console {
		tabCon = theme.TabActive
	} else {
		tabLog = theme.TabActive
	}
	right := tabLog.Render("Log") + tabCon.Render("Console") + "  " + theme.KeyHint.Render(keyLabel("ctrl+k")) + theme.KeyLabel.Render(" commands  ") + theme.KeyHint.Render("?") + theme.KeyLabel.Render(" help ")
	// Image logos occupy cells the width calculation cannot see.
	return highlight(joinRow(" "+left, right, m.width-logoHidden()), theme.Surface)
}

func (m *Model) hasFilter() bool {
	return m.logOpts.Grep != "" || m.logOpts.Author != "" || m.logOpts.Ref != "" || len(m.logOpts.Paths) > 0 || !m.logOpts.All
}

func (m *Model) renderFilterBar() string {
	var sb strings.Builder
	sb.WriteString(" " + logoFilterPrefix())
	if m.searching {
		sb.WriteString(theme.FilterInput.Render("⌕ " + m.search.View()))
	} else {
		text := m.logOpts.Grep
		st := theme.FilterChip
		if text == "" {
			text = searchPlaceholder
		} else {
			st = theme.FilterChipActive
		}
		sb.WriteString(st.Render("⌕ " + trunc(text, 36)))
	}
	sb.WriteString("  ")

	branchSt := theme.FilterChip
	if m.logOpts.Ref != "" || !m.logOpts.All {
		branchSt = theme.FilterChipActive
	}
	if m.barBranch {
		branchSt = theme.FilterChipFocus
	}
	sb.WriteString(branchSt.Render(m.branchLabel()) + "  ")

	if len(m.logOpts.Paths) > 0 {
		sb.WriteString(theme.FilterChipActive.Render("Path: "+trunc(m.logOpts.Paths[0], 30)) + "  ")
	}
	right := ""
	switch {
	case m.barBranch:
		right = keyHints("enter", "choose branch", "esc", "clear", "←", "search", "↓", "log") + " "
	case m.searching:
		right = keyHints("enter", "apply", "→", "branch", "esc", "cancel") + " "
	case m.hasFilter():
		right = theme.KeyHint.Render("esc") + theme.KeyLabel.Render(" clear filters ")
	}
	return highlight(joinRow(sb.String(), right, m.width), theme.Surface)
}

func (m *Model) renderPanels() string {
	m.layout() // rects depend on the mode (commit view widens the diff)
	bl := m.rects[PanelBranches]
	lg := m.rects[PanelLog]
	ch := m.rects[PanelChanges]
	dt := m.detailsRect

	branches := frame("Branches", "", m.renderBranches(bl.w, bl.h), bl.w, bl.h, m.focus == PanelBranches)

	if m.commitOpen {
		center := frame(m.commitTitle(), plural(len(m.status), "file", "files"), m.renderCommit(lg.w, lg.h), lg.w, lg.h, m.commit == nil || m.commit.focus != cfDiff)
		dr := m.diffRect
		diffActive := m.commit != nil && m.commit.focus == cfDiff
		var right string
		if m.diff != nil {
			right = frame(m.diffTitle(), "", m.renderDiff(dr.w, dr.h), dr.w, dr.h, diffActive)
		} else {
			right = frame("Diff", "", []string{"", theme.MutedSt.Render("  Select a file to preview its changes")}, dr.w, dr.h, diffActive)
		}
		return lipgloss.JoinHorizontal(lipgloss.Top, branches, center, right)
	}

	if m.conflictOpen && m.conflict != nil {
		center := frame(m.conflictTitle(), "", m.renderConflict(lg.w, lg.h), lg.w, lg.h, true)
		files := frame("Conflicted files", fmt.Sprintf("%d", len(m.conflict.files)), m.renderConflictFiles(ch.w), ch.w, ch.h, false)
		help := frame("Keys", "", m.renderConflictHelp(dt.w), dt.w, dt.h, false)
		right := lipgloss.JoinVertical(lipgloss.Left, files, help)
		return lipgloss.JoinHorizontal(lipgloss.Top, branches, center, right)
	}

	var center string
	switch {
	case m.rebaseOpen && m.rebase != nil:
		center = frame(m.rebaseTitle(), "", m.renderRebase(lg.w, lg.h), lg.w, lg.h, m.focus == PanelLog)
	case m.console:
		center = frame("Console", plural(len(m.repo.History()), "command", "commands"), m.renderConsole(lg.w, lg.h), lg.w, lg.h, m.focus == PanelLog)
	case m.diff != nil:
		center = frame(m.diffTitle(), "esc to close", m.renderDiff(lg.w, lg.h), lg.w, lg.h, m.focus == PanelLog)
	default:
		count := fmt.Sprintf("%d", len(m.commits))
		if m.logTruncated {
			count += "+"
		}
		center = frame("Log", count, m.renderLog(lg.w, lg.h), lg.w, lg.h, m.focus == PanelLog)
	}

	filesTitle := "Changes"
	if m.filesFor == "local" {
		filesTitle = "Local Changes"
	}
	files := frame(filesTitle, fmt.Sprintf("%d", len(m.files)), m.renderFiles(ch.w, ch.h), ch.w, ch.h, m.focus == PanelChanges)
	details := frame("Details", "", m.renderDetails(dt.w, dt.h), dt.w, dt.h, m.detailsFocus && m.focus == PanelChanges)
	right := lipgloss.JoinVertical(lipgloss.Left, files, details)
	return lipgloss.JoinHorizontal(lipgloss.Top, branches, center, right)
}

// frame draws a rounded border with an inline title around content lines.
// w and h are the inner dimensions.
func frame(title, count string, lines []string, w, h int, active bool) string {
	border := theme.FrameBorder
	titleSt := theme.FrameTitle
	if active {
		border = theme.FrameBorderActive
		titleSt = theme.FrameTitleActive
	}
	t := " " + titleSt.Render(title) + " "
	if count != "" {
		t += theme.FrameCount.Render(count) + " "
	}
	tw := width(t)
	if tw > w-2 {
		t = trunc(t, w-2)
		tw = width(t)
	}
	top := border.Render("╭─") + t + border.Render(strings.Repeat("─", maxInt(0, w-1-tw))+"╮")
	bottom := border.Render("╰" + strings.Repeat("─", w) + "╯")
	side := border.Render("│")
	var sb strings.Builder
	sb.WriteString(top + "\n")
	for i := 0; i < h; i++ {
		var l string
		if i < len(lines) {
			l = pad(lines[i], w)
		} else {
			l = strings.Repeat(" ", w)
		}
		sb.WriteString(side + l + side + "\n")
	}
	sb.WriteString(bottom)
	return sb.String()
}

func (m *Model) renderStatusBar() string {
	var hints string
	switch {
	case m.menu != nil:
		hints = keyHints("↑↓", "navigate", "enter", "select", "esc", "close")
	case m.dialog != nil:
		hints = keyHints("enter", "confirm", "esc", "cancel")
	case m.searching:
		hints = keyHints("enter", "apply", "esc", "cancel")
	case m.diff != nil:
		hints = keyHints("↑↓", "next/prev change", "shift+↑↓", "line", "space", "page", "n/p", "next/prev file", "esc", "back")
	case m.console:
		hints = keyHints("j/k", "scroll", "`", "back to log")
	case m.focus == PanelBranches:
		hints = keyHints(keyLabel("ctrl+k"), "palette", "enter", "actions", "c", "checkout", "C", "commit", "u", "undo", "P", "push", "p", "pull", "f", "fetch", "v", "version tag", "←→", "fold/section")
	case m.detailsFocus:
		hints = keyHints("j/k", "scroll", "↑", "back to files", "esc", "back")
	case m.focus == PanelLog:
		hints = keyHints(keyLabel("ctrl+k"), "palette", "enter", "actions", "c", "commit", "i", "rebase", "u", "undo", "P", "push", "p", "pull", "f", "fetch", "v", "version tag", "/", "search", "A", "all/head", "y", "copy hash", "←→", "section")
	default:
		if m.filesFor == "local" && m.hasConflicts() {
			hints = keyHints("x", "resolve conflicts", "enter", "resolve file", keyLabel("ctrl+k"), "palette", "c", "commit", "←→", "fold/section")
		} else if m.filesFor == "local" {
			hints = keyHints(keyLabel("ctrl+k"), "palette", "space", "check", "c", "commit", "P", "push", "enter", "diff", "d", "discard", "H", "history", "←→", "fold/section")
		} else {
			hints = keyHints(keyLabel("ctrl+k"), "palette", "enter", "diff", "c", "commit", "H", "history", "y", "copy path", "space", "fold", "←→", "fold/section")
		}
	}
	switch {
	case m.push != nil:
		hints = keyHints("enter", "push", "f", "force with lease", "t", "tags", "esc", "cancel")
	case m.commitOpen && m.commit != nil && m.commit.focus == cfMessage:
		hints = keyHints(keyLabel("ctrl+s"), "commit", keyLabel("ctrl+p"), "commit & push", "↑", "history", "tab", "buttons", "esc", "files")
	case m.rebaseOpen:
		hints = keyHints("p r e s f d", "action", "⇧↑↓", "reorder", "enter", "menu", keyLabel("ctrl+s"), "start", "esc", "cancel")
	case m.conflictOpen:
		hints = keyHints("o", "ours", "t", "theirs", "b", "both", "↑↓", "conflict", "n/p", "file", keyLabel("ctrl+s"), "save & resolve", "esc", "back")
	case m.commitOpen && m.commit != nil && m.commit.focus == cfDiff:
		hints = keyHints("space", "check hunk", "a", "whole file", "↑↓", "hunk", "n/p", "file", "shift+↑↓", "line", "←/esc", "files")
	case m.commitOpen:
		hints = keyHints("enter", "check", "a", "check all", "tab", "message", "→", "diff", "1 2 3", "pane", "d", "discard", keyLabel("ctrl+s"), "commit", "esc", "back")
	}
	quit := ""
	if m.menu == nil && m.dialog == nil && m.push == nil {
		quit = theme.QuitKey.Render("q") + " " + theme.KeyLabel.Render("exit") + theme.DimSt.Render("  ")
	}
	right := ""
	if m.loading > 0 {
		right = m.spin.View() + theme.MutedSt.Render(" loading…")
	} else {
		right = theme.DimSt.Render(fmt.Sprintf("%d/%d", minInt(m.lcur+1, m.logLen()), m.logLen()))
	}
	return highlight(joinRow(" "+hints, quit+right+" ", m.width), theme.Surface)
}

func (m *Model) renderHelp() string {
	type row struct{ k, d string }
	type section struct {
		title string
		rows  []row
	}
	sections := []section{
		{"Navigation", []row{{"tab / 1 2 3 / h l", "switch panel"}, {"j k ↑ ↓", "move"}, {"g / G", "top / bottom"}, {keyLabel("ctrl+d") + " / " + keyLabel("ctrl+u"), "half page"}, {"space", "fold / unfold tree"}, {"← →", "fold / unfold, then previous / next pane"}, {"mouse", "click · right-click · wheel"}}},
		{"Log", []row{{"enter / m / right-click", "commit actions"}, {"i", "interactive rebase from this commit"}, {"✓ ✗ ◌", "CI status (GitHub, via gh token)"}, {"/", "search message, author or hash"}, {"↑ at top", "jump to the search bar"}, {"→ / enter (filter bar)", "branch picker · esc clears"}, {"A", "all branches ↔ current"}, {"y", "copy hash"}, {"esc", "clear filters"}}},
		{"Branches", []row{{"enter / m", "branch actions"}, {"c", "checkout"}, {"s", "show branch in log"}, {"d", "delete"}, {"f / p / P", "fetch / pull / push"}}},
		{"Changes", []row{{"enter", "open diff"}, {"↑ ↓ (in diff)", "next / previous change block"}, {"shift+↑ ↓", "scroll one line"}, {"n / p", "next / prev file (in diff)"}, {"space / a", "check file / all (local)"}, {"c / C", "commit workspace"}, {"d", "discard (local)"}, {"H", "file history in log"}}},
		{"Commit & Push", []row{{keyLabel("ctrl+s"), "commit selected files"}, {keyLabel("ctrl+p"), "commit & push"}, {"space (in diff)", "check / uncheck a hunk"}, {"↑ (in message)", "previous messages"}, {"P", "push dialog"}, {"p", "pull (merge / rebase / fetch)"}}},
		{"Other", []row{{keyLabel("ctrl+k"), "command palette — search every action"}, {"x", "resolve merge conflicts (ours / theirs / both per block)"}, {"u", "undo the last commit / rebase / reset / checkout / deletion"}, {"v", "new version tag (patch / minor / major) and push it"}, {"`", "console (git commands)"}, {"r", "refresh"}, {"?", "this help"}, {"q", "quit"}}},
	}
	renderSection := func(s section) []string {
		lines := []string{theme.Bold.Render(s.title)}
		for _, r := range s.rows {
			lines = append(lines, "  "+pad(theme.KeyHint.Render(r.k), 26)+theme.MutedSt.Render(r.d))
		}
		return append(lines, "")
	}
	var cols [][]string
	if m.width >= 120 && m.height < 44 {
		var left, right []string
		for i, s := range sections {
			if i < 2 {
				left = append(left, renderSection(s)...)
			} else {
				right = append(right, renderSection(s)...)
			}
		}
		cols = [][]string{left, right}
	} else {
		var all []string
		for _, s := range sections {
			all = append(all, renderSection(s)...)
		}
		cols = [][]string{all}
	}
	colW := 0
	for _, c := range cols {
		for _, l := range c {
			colW = maxInt(colW, width(l))
		}
	}
	rows := 0
	for _, c := range cols {
		rows = maxInt(rows, len(c))
	}
	var lines []string
	lines = append(lines, theme.AccentB.Render("gitpad")+theme.MutedSt.Render(" — keyboard reference"), "")
	for i := 0; i < rows; i++ {
		var parts []string
		for _, c := range cols {
			l := ""
			if i < len(c) {
				l = c[i]
			}
			parts = append(parts, pad(l, colW))
		}
		lines = append(lines, strings.Join(parts, "   "))
	}
	lines = append(lines, theme.DimSt.Render("esc / ? to close"))
	// Never exceed the screen.
	if maxL := m.height - 4; len(lines) > maxL && maxL > 3 {
		lines = append(lines[:maxL-1], theme.DimSt.Render("… (enlarge the terminal to see everything)"))
	}
	return theme.DialogBox.Render(strings.Join(lines, "\n"))
}

func (m *Model) renderConsole(w, h int) []string {
	hist := m.repo.History()
	var all []string
	for _, e := range hist {
		st := lipgloss.NewStyle().Foreground(theme.Accent)
		if e.Err != nil {
			st = lipgloss.NewStyle().Foreground(theme.Red)
		}
		head := theme.DimSt.Render(e.At.Format("15:04:05")) + " " + st.Render("$ git "+strings.Join(e.Args, " ")) + theme.DimSt.Render(fmt.Sprintf("  %dms", e.Took.Milliseconds()))
		all = append(all, trunc(head, w))
		out := strings.TrimRight(e.Output, "\n")
		if out != "" {
			ol := strings.Split(out, "\n")
			max := 8
			for i, l := range ol {
				if i >= max {
					all = append(all, theme.DimSt.Render(fmt.Sprintf("    … +%d lines", len(ol)-max)))
					break
				}
				all = append(all, theme.MutedSt.Render(trunc("    "+l, w)))
			}
		}
	}
	if len(all) == 0 {
		return []string{"", theme.MutedSt.Render("  No commands executed yet")}
	}
	maxScroll := maxInt(0, len(all)-h)
	m.cscroll = clamp(m.cscroll, 0, maxScroll)
	end := len(all) - m.cscroll
	start := maxInt(0, end-h)
	return all[start:end]
}
