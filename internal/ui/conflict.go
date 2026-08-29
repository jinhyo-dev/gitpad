package ui

import (
	"fmt"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/jinhyo-dev/gitpad/internal/git"
	"github.com/jinhyo-dev/gitpad/internal/ui/theme"
)

// The conflict workspace replaces the log pane while a merge / rebase /
// cherry-pick has unmerged files: the file is shown with its conflict blocks
// highlighted, each block is resolved with a keystroke, and saving writes the
// file and marks it resolved (git add).

type conflictState struct {
	files   []string // conflicted paths
	fi      int      // current file
	lines   []string
	blocks  []git.Conflict
	choices []git.Choice
	cur     int // current block
	scroll  int
	err     error
}

type conflictFilesMsg struct {
	files []string
	err   error
}
type conflictFileMsg struct {
	path    string
	content string
	err     error
}

const (
	glyphOurs   = "◀"
	glyphTheirs = "▶"
)

func (m *Model) conflictCount() int {
	n := 0
	for _, f := range m.status {
		if f.Conflict {
			n++
		}
	}
	return n
}

func (m *Model) hasConflicts() bool { return m.conflictCount() > 0 }

// openConflicts loads the conflicted file list and opens the workspace.
func (m *Model) openConflicts(prefer string) tea.Cmd {
	m.loading++
	repo := m.repo
	return func() tea.Msg {
		files, err := repo.ConflictedFiles()
		if err == nil && prefer != "" {
			for i, f := range files {
				if f == prefer && i > 0 {
					files = append([]string{f}, append(files[:i:i], files[i+1:]...)...)
					break
				}
			}
		}
		return conflictFilesMsg{files: files, err: err}
	}
}

func (m *Model) onConflictFiles(msg conflictFilesMsg) tea.Cmd {
	m.loading--
	if msg.err != nil {
		return m.showToast(msg.err.Error(), 2)
	}
	if len(msg.files) == 0 {
		m.closeConflicts()
		if m.info.State != "" {
			return m.offerContinue()
		}
		return m.showToast("No conflicted files", 0)
	}
	if m.conflict == nil {
		m.conflict = &conflictState{}
	}
	m.conflict.files = msg.files
	m.conflict.fi = 0
	m.conflictOpen = true
	m.console, m.diff = false, nil
	m.focus = PanelLog
	return m.loadConflictFile()
}

func (m *Model) closeConflicts() {
	m.conflictOpen = false
	m.conflict = nil
}

func (m *Model) loadConflictFile() tea.Cmd {
	c := m.conflict
	if c == nil || c.fi >= len(c.files) {
		return nil
	}
	m.loading++
	repo := m.repo
	path := c.files[c.fi]
	return func() tea.Msg {
		content, err := repo.ReadWorktreeFile(path)
		return conflictFileMsg{path: path, content: content, err: err}
	}
}

func (m *Model) onConflictFile(msg conflictFileMsg) {
	m.loading--
	c := m.conflict
	if c == nil || c.fi >= len(c.files) || c.files[c.fi] != msg.path {
		return
	}
	c.err = msg.err
	c.lines, c.blocks = git.ParseConflicts(msg.content)
	c.choices = make([]git.Choice, len(c.blocks))
	c.cur, c.scroll = 0, 0
	if len(c.blocks) > 0 {
		c.scroll = maxInt(0, c.blocks[0].Start-2)
	}
}

// offerContinue is shown once every conflict is resolved.
func (m *Model) offerContinue() tea.Cmd {
	state := m.info.State
	if state == "" {
		return nil
	}
	m.confirm("All conflicts resolved", "Continue the "+state+" now?", "Continue", false, func(m *Model) tea.Cmd {
		return m.action("Continue "+state, func() error { return m.repo.ContinueState(state) })
	})
	return nil
}

func (m *Model) conflictTitle() string {
	c := m.conflict
	if c == nil || c.fi >= len(c.files) {
		return "Resolve conflicts"
	}
	done := 0
	for _, ch := range c.choices {
		if ch != git.Unresolved {
			done++
		}
	}
	return fmt.Sprintf("Resolve · %s · conflict %d/%d · %d resolved", c.files[c.fi], minInt(c.cur+1, len(c.blocks)), len(c.blocks), done)
}

// conflictMaxScroll bounds scrolling.
func (m *Model) conflictMaxScroll() int {
	return maxInt(0, len(m.conflict.lines)-m.rects[PanelLog].h)
}

func (m *Model) renderConflict(w, h int) []string {
	c := m.conflict
	var out []string
	if c.err != nil {
		return []string{theme.MenuDanger.Render(trunc("  "+c.err.Error(), w))}
	}
	if len(c.blocks) == 0 {
		return []string{"", theme.MutedSt.Render("  No conflict markers in this file — save (ctrl+s) to mark it resolved.")}
	}
	oursBg := lipgloss.AdaptiveColor{Light: "#e1ecff", Dark: "#1f2b45"}
	theirsBg := lipgloss.AdaptiveColor{Light: "#f1e4fb", Dark: "#33254a"}
	baseBg := lipgloss.AdaptiveColor{Light: "#ececec", Dark: "#2a2a33"}
	gutter := 5
	// Map each line to its block & section.
	type mark struct {
		block   int
		section byte // 'o' ours, 'b' base, 't' theirs, 'h' header/marker
	}
	marks := map[int]mark{}
	for bi, b := range c.blocks {
		for i := b.Start; i <= b.End; i++ {
			l := c.lines[i]
			switch {
			case i == b.Start || i == b.End || strings.HasPrefix(l, "=======") || strings.HasPrefix(l, "|||||||"):
				marks[i] = mark{bi, 'h'}
			}
		}
		pos := b.Start + 1
		for range b.Ours {
			marks[pos] = mark{bi, 'o'}
			pos++
		}
		if b.Base != nil {
			pos++ // ||||||| line
			for range b.Base {
				marks[pos] = mark{bi, 'b'}
				pos++
			}
		}
		pos++ // ======= line
		for range b.Theirs {
			marks[pos] = mark{bi, 't'}
			pos++
		}
	}
	end := minInt(len(c.lines), c.scroll+h)
	for i := c.scroll; i < end; i++ {
		l := strings.ReplaceAll(c.lines[i], "\t", "    ")
		mk, inBlock := marks[i]
		num := theme.DiffGutter.Render(padLeft(strconv.Itoa(i+1), gutter))
		var line string
		if !inBlock {
			line = pad(num+"  "+theme.Base.Render(trunc(l, w-gutter-3)), w-1)
		} else {
			b := c.blocks[mk.block]
			ch := c.choices[mk.block]
			switch mk.section {
			case 'h':
				var label string
				var st lipgloss.Style
				switch {
				case i == b.Start:
					label = fmt.Sprintf("%s ours · %s", glyphOurs, orDefault(b.OursLabel, "HEAD"))
					st = lipgloss.NewStyle().Foreground(theme.Blue).Bold(true)
				case i == b.End:
					label = fmt.Sprintf("%s theirs · %s", glyphTheirs, orDefault(b.TheirsLabel, "incoming"))
					st = lipgloss.NewStyle().Foreground(theme.Magenta).Bold(true)
				case strings.HasPrefix(l, "|||||||"):
					label = "base"
					st = theme.DimSt
				default:
					label = "═══"
					st = theme.DimSt
				}
				state := ""
				switch ch {
				case git.KeepOurs:
					state = "  ✓ keep ours"
				case git.KeepTheirs:
					state = "  ✓ keep theirs"
				case git.KeepBoth:
					state = "  ✓ keep both (ours first)"
				case git.KeepBothReverse:
					state = "  ✓ keep both (theirs first)"
				}
				if i == b.Start && state != "" {
					label += lipgloss.NewStyle().Foreground(theme.Green).Bold(true).Render(state)
				}
				line = pad(num+"  "+st.Render(label), w-1)
			default:
				bg, fg := oursBg, theme.Blue
				if mk.section == 't' {
					bg, fg = theirsBg, theme.Magenta
				} else if mk.section == 'b' {
					bg, fg = baseBg, theme.Muted
				}
				// Dim what the current choice throws away.
				dropped := (ch == git.KeepOurs && mk.section == 't') || (ch == git.KeepTheirs && mk.section == 'o') || (ch != git.Unresolved && mk.section == 'b')
				st := lipgloss.NewStyle().Foreground(fg)
				if dropped {
					st = theme.DimSt.Strikethrough(true)
				}
				body := pad(num+"  "+st.Render(trunc(l, w-gutter-3)), w-1)
				if !dropped {
					body = highlight(body, bg)
				}
				line = body
			}
		}
		marker := " "
		if inBlock && mk.block == c.cur {
			marker = theme.AccentB.Render("▎")
		}
		out = append(out, marker+line)
	}
	return out
}

func (m *Model) conflictJump(dir int) {
	c := m.conflict
	if len(c.blocks) == 0 {
		return
	}
	c.cur = clamp(c.cur+dir, 0, len(c.blocks)-1)
	c.scroll = clamp(c.blocks[c.cur].Start-2, 0, m.conflictMaxScroll())
}

func (m *Model) conflictKey(k tea.KeyMsg) tea.Cmd {
	c := m.conflict
	key := k.String()
	choose := func(ch git.Choice) tea.Cmd {
		if c.cur < len(c.choices) {
			c.choices[c.cur] = ch
			if ch != git.Unresolved && c.cur < len(c.blocks)-1 {
				m.conflictJump(1)
			}
		}
		return nil
	}
	switch key {
	case "esc", "q":
		m.closeConflicts()
		return nil
	case "ctrl+s", "enter":
		return m.saveConflictFile()
	case "j", "down":
		m.conflictJump(1)
	case "k", "up":
		m.conflictJump(-1)
	case "shift+down", "ctrl+e":
		c.scroll = minInt(c.scroll+1, m.conflictMaxScroll())
	case "shift+up", "ctrl+y":
		c.scroll = maxInt(c.scroll-1, 0)
	case "ctrl+d", "pgdown", " ":
		c.scroll = minInt(c.scroll+m.rects[PanelLog].h/2, m.conflictMaxScroll())
	case "ctrl+u", "pgup":
		c.scroll = maxInt(c.scroll-m.rects[PanelLog].h/2, 0)
	case "g", "home":
		c.cur, c.scroll = 0, 0
	case "G", "end":
		c.cur = maxInt(0, len(c.blocks)-1)
		c.scroll = m.conflictMaxScroll()
	case "o":
		return choose(git.KeepOurs)
	case "t":
		return choose(git.KeepTheirs)
	case "b":
		return choose(git.KeepBoth)
	case "B":
		return choose(git.KeepBothReverse)
	case "x":
		return choose(git.Unresolved)
	case "O":
		for i := range c.choices {
			c.choices[i] = git.KeepOurs
		}
	case "T":
		for i := range c.choices {
			c.choices[i] = git.KeepTheirs
		}
	case "n":
		return m.conflictStep(1)
	case "p":
		return m.conflictStep(-1)
	case "e":
		return m.showToast("Open the file in your editor, then save here to mark it resolved", 0)
	}
	return nil
}

func (m *Model) conflictStep(dir int) tea.Cmd {
	c := m.conflict
	nx := c.fi + dir
	if nx < 0 || nx >= len(c.files) {
		return nil
	}
	c.fi = nx
	return m.loadConflictFile()
}

// saveConflictFile writes the resolution and marks the file resolved.
func (m *Model) saveConflictFile() tea.Cmd {
	c := m.conflict
	if c == nil || c.fi >= len(c.files) {
		return nil
	}
	content, all := git.Resolve(c.lines, c.blocks, c.choices)
	if !all {
		left := 0
		for _, ch := range c.choices {
			if ch == git.Unresolved {
				left++
			}
		}
		return m.showToast(fmt.Sprintf("%s still unresolved — choose o / t / b for each", plural(left, "conflict", "conflicts")), 2)
	}
	path := c.files[c.fi]
	repo := m.repo
	return m.actionThen("Resolve "+path, func() error {
		if err := repo.WriteWorktreeFile(path, content); err != nil {
			return err
		}
		return repo.StagePath(path)
	}, func(m *Model) tea.Cmd {
		// Reload the remaining conflicted files (drops the one just saved).
		return m.openConflicts("")
	})
}

// conflictMouse selects blocks by click and scrolls with the wheel.
func (m *Model) conflictMouse(msg tea.MouseMsg) tea.Cmd {
	c := m.conflict
	rect := m.rects[PanelLog]
	if !rect.contains(msg.X, msg.Y) {
		return nil
	}
	switch msg.Button {
	case tea.MouseButtonWheelDown:
		c.scroll = minInt(c.scroll+3, m.conflictMaxScroll())
		return nil
	case tea.MouseButtonWheelUp:
		c.scroll = maxInt(c.scroll-3, 0)
		return nil
	}
	if msg.Action != tea.MouseActionPress {
		return nil
	}
	line := c.scroll + msg.Y - rect.y
	for i, b := range c.blocks {
		if line >= b.Start && line <= b.End {
			c.cur = i
			break
		}
	}
	return nil
}

// renderConflictFiles is the right-column list of conflicted paths.
func (m *Model) renderConflictFiles(w int) []string {
	c := m.conflict
	var out []string
	for i, f := range c.files {
		row := " " + lipgloss.NewStyle().Foreground(theme.Red).Bold(true).Render("U") + " " + theme.Base.Render(trunc(f, w-4))
		if i == c.fi {
			row = highlight(pad(row, w), theme.Selection)
		}
		out = append(out, pad(row, w))
	}
	return out
}

func (m *Model) renderConflictHelp(w int) []string {
	rows := [][2]string{
		{"o / t", "keep ours / theirs"},
		{"b / B", "keep both (ours first / theirs first)"},
		{"O / T", "ours / theirs for every conflict"},
		{"x", "clear the choice"},
		{"↑ ↓", "next / previous conflict"},
		{"n / p", "next / previous file"},
		{keyLabel("ctrl+s"), "save file & mark resolved"},
		{"esc", "back to the log"},
	}
	var out []string
	out = append(out, pad(" "+theme.Bold.Render("Resolving a "+orDefault(m.info.State, "merge")), w))
	out = append(out, pad("", w))
	for _, r := range rows {
		out = append(out, pad(" "+pad(theme.KeyHint.Render(r[0]), 10)+theme.MutedSt.Render(trunc(r[1], w-12)), w))
	}
	return out
}
