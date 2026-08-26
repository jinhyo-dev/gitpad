package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/jinhyo/gitpad/internal/git"
	"github.com/jinhyo/gitpad/internal/ui/theme"
)

// The commit workspace replaces the log pane: a file checklist on top, a
// multi-line message editor below, and Commit / Commit & Push buttons. The
// right column shows the diff of the highlighted file.

type commitFocus int

const (
	cfFiles commitFocus = iota
	cfMessage
	cfButtons
)

type commitState struct {
	focus   commitFocus
	msg     textarea.Model
	history []string
	histIdx int // -1 = editing the draft
	draft   string
	button  int // 0 commit, 1 commit & push
	diffSeq int
}

type historyMsg struct{ msgs []string }
type commitDiffMsg struct{ seq int }

const (
	commitMsgMinH = 4
	commitMsgMaxH = 8
)

func newCommitState(width int) *commitState {
	ta := textarea.New()
	ta.Prompt = "┃ "
	ta.Placeholder = "Commit message — first line is the subject"
	ta.ShowLineNumbers = false
	ta.EndOfBufferCharacter = ' '
	ta.CharLimit = 0
	ta.KeyMap.LinePrevious = key.NewBinding(key.WithKeys("up"))
	ta.KeyMap.LineNext = key.NewBinding(key.WithKeys("down"))
	ta.KeyMap.DeleteAfterCursor = key.NewBinding(key.WithKeys("ctrl+k"))
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.FocusedStyle.Prompt = lipgloss.NewStyle().Foreground(theme.Accent)
	ta.FocusedStyle.Text = lipgloss.NewStyle().Foreground(theme.Text)
	ta.FocusedStyle.Placeholder = lipgloss.NewStyle().Foreground(theme.Dim)
	ta.FocusedStyle.EndOfBuffer = lipgloss.NewStyle()
	ta.BlurredStyle = ta.FocusedStyle
	ta.BlurredStyle.Prompt = lipgloss.NewStyle().Foreground(theme.Dim)
	ta.Cursor.Style = lipgloss.NewStyle().Foreground(theme.Accent)
	ta.SetWidth(width)
	ta.SetHeight(commitMsgMinH)
	return &commitState{msg: ta, histIdx: -1}
}

// commitLayout splits the center pane into files / message / buttons.
func (m *Model) commitLayout() (files, msg, btn rect) {
	r := m.rects[PanelLog]
	msgH := clamp(r.h/3, commitMsgMinH, commitMsgMaxH)
	filesH := r.h - msgH - 3 // two separators + button row
	if filesH < 3 {
		filesH = 3
		msgH = maxInt(2, r.h-filesH-3)
	}
	files = rect{x: r.x, y: r.y, w: r.w, h: filesH}
	msg = rect{x: r.x, y: r.y + filesH + 1, w: r.w, h: msgH}
	btn = rect{x: r.x, y: r.y + filesH + 1 + msgH + 1, w: r.w, h: 1}
	return
}

// openCommit enters the commit workspace.
func (m *Model) openCommit() tea.Cmd {
	if len(m.status) == 0 {
		return m.showToast("Nothing to commit — working tree clean", 0)
	}
	_, msgRect, _ := m.commitLayout()
	if m.commit == nil {
		m.commit = newCommitState(msgRect.w)
	}
	m.commitOpen = true
	m.console = false
	m.focus = PanelLog
	m.layout()
	_, msgRect, _ = m.commitLayout()
	m.commit.focus = cfFiles
	m.commit.msg.Blur()
	m.commit.msg.SetWidth(msgRect.w)
	m.commit.msg.SetHeight(msgRect.h)
	m.lcur, m.lscroll = 0, 0 // the "Uncommitted changes" row
	var cmds []tea.Cmd
	if m.filesFor != "local" {
		m.fcur, m.fscroll = 0, 0
		cmds = append(cmds, m.loadFiles(nil))
	} else {
		cmds = append(cmds, m.commitDiffNow())
	}
	if len(m.commit.history) == 0 {
		repo := m.repo
		cmds = append(cmds, func() tea.Msg { return historyMsg{msgs: repo.RecentMessages(20)} })
	}
	return tea.Batch(cmds...)
}

func (m *Model) closeCommit() {
	m.commitOpen = false
	m.diff = nil
	m.layout()
	if m.commit != nil {
		m.commit.msg.Blur()
	}
}

// commitDiffNow shows the diff of the file under the cursor immediately.
func (m *Model) commitDiffNow() tea.Cmd {
	n := m.selectedFileNode()
	if n == nil || n.isDir {
		m.diff = nil
		return nil
	}
	if m.diff != nil && m.diff.path == n.file.Path && !m.diff.loading {
		return nil
	}
	return m.openDiff(*n.file)
}

// commitDiffLater debounces diff loading while the cursor is moving.
func (m *Model) commitDiffLater() tea.Cmd {
	m.commit.diffSeq++
	seq := m.commit.diffSeq
	return tea.Tick(selectDebounce, func(time.Time) tea.Msg { return commitDiffMsg{seq: seq} })
}

// ---- selection ---------------------------------------------------------

// syncSelection keeps the checkbox map aligned with the working tree:
// tracked changes are checked by default, unversioned files are not.
func (m *Model) syncSelection() {
	if m.selected == nil {
		m.selected = map[string]bool{}
	}
	present := map[string]bool{}
	for _, f := range m.status {
		present[f.Path] = true
		if _, ok := m.selected[f.Path]; !ok {
			m.selected[f.Path] = f.Status != '?'
		}
	}
	for p := range m.selected {
		if !present[p] {
			delete(m.selected, p)
		}
	}
}

func (m *Model) selectedFiles() []git.FileChange {
	var out []git.FileChange
	for _, f := range m.status {
		if m.selected[f.Path] {
			out = append(out, f)
		}
	}
	return out
}

// nodePaths lists the working-tree paths under a group / directory node.
func (m *Model) nodePaths(n *fnode) []string {
	if !n.isDir {
		return []string{n.file.Path}
	}
	untracked := n.key == "g:unversioned"
	group := ""
	if strings.HasPrefix(n.key, "g:") {
		group = strings.SplitN(n.key, "/", 2)[0]
		untracked = group == "g:unversioned"
	}
	dir := strings.TrimPrefix(strings.TrimPrefix(n.key, group), "/")
	var paths []string
	for _, f := range m.status {
		if group != "" && (f.Status == '?') != untracked {
			continue
		}
		if !n.isGroup && !strings.HasPrefix(f.Path, dir+"/") {
			continue
		}
		paths = append(paths, f.Path)
	}
	return paths
}

// toggleNodeSelection toggles a file, or every file below a group/dir.
func (m *Model) toggleNodeSelection(n *fnode) {
	if !n.isDir {
		m.selected[n.file.Path] = !m.selected[n.file.Path]
		return
	}
	paths := m.nodePaths(n)
	all := true
	for _, p := range paths {
		if !m.selected[p] {
			all = false
			break
		}
	}
	for _, p := range paths {
		m.selected[p] = !all
	}
}

func (m *Model) toggleAllSelection() {
	all := true
	for _, f := range m.status {
		if !m.selected[f.Path] {
			all = false
			break
		}
	}
	for _, f := range m.status {
		m.selected[f.Path] = !all
	}
}

// ---- rendering ---------------------------------------------------------

func (m *Model) renderCommit(w, h int) []string {
	filesRect, msgRect, _ := m.commitLayout()
	c := m.commit
	var out []string

	// Files.
	files := m.renderFiles(w, filesRect.h)
	for len(files) < filesRect.h {
		files = append(files, strings.Repeat(" ", w))
	}
	out = append(out, files[:filesRect.h]...)

	// Message.
	sel := len(m.selectedFiles())
	title := theme.FrameTitle.Render("Commit message")
	if c.focus == cfMessage {
		title = theme.FrameTitleActive.Render("Commit message")
	}
	hist := ""
	if c.histIdx >= 0 {
		hist = theme.DimSt.Render(fmt.Sprintf("history %d/%d  ", c.histIdx+1, len(c.history)))
	} else if len(c.history) > 0 {
		hist = theme.DimSt.Render("↑ history  ")
	}
	out = append(out, separator(w, " "+title+" ", hist))
	msgLines := strings.Split(c.msg.View(), "\n")
	for i := 0; i < msgRect.h; i++ {
		l := ""
		if i < len(msgLines) {
			l = msgLines[i]
		}
		out = append(out, pad(l, w))
	}
	out = append(out, separator(w, "", ""))

	// Buttons.
	commitBtn, pushBtn := theme.ButtonPlain, theme.ButtonPlain
	if c.focus == cfButtons {
		if c.button == 0 {
			commitBtn = theme.ButtonPrimary
		} else {
			pushBtn = theme.ButtonPrimary
		}
	} else {
		commitBtn = theme.ButtonPrimary
	}
	btns := " " + commitBtn.Render("Commit") + "  " + pushBtn.Render("Commit & Push")
	count := theme.MutedSt.Render(fmt.Sprintf("%d/%d files ", sel, len(m.status)))
	hint := theme.KeyHint.Render("ctrl+s") + theme.KeyLabel.Render(" commit  ") + theme.KeyHint.Render("ctrl+p") + theme.KeyLabel.Render(" & push  ")
	info := hint + count
	if width(btns)+width(info) > w { // buttons always win
		info = count
	}
	out = append(out, joinRow(btns, info, w))
	return out
}

func separator(w int, left, right string) string {
	line := theme.FrameBorder.Render(strings.Repeat("─", w))
	if left == "" && right == "" {
		return line
	}
	lw, rw := width(left), width(right)
	mid := maxInt(0, w-lw-rw)
	return left + theme.FrameBorder.Render(strings.Repeat("─", mid)) + right
}

func (m *Model) commitTitle() string {
	return "Commit → " + m.currentBranch()
}

// ---- input -------------------------------------------------------------

func (m *Model) commitKey(k tea.KeyMsg) tea.Cmd {
	c := m.commit
	key := k.String()
	switch key {
	case "ctrl+s":
		return m.doCommit(false)
	case "ctrl+p":
		return m.doCommit(true)
	case "tab":
		return m.commitSetFocus((c.focus + 1) % 3)
	case "shift+tab":
		return m.commitSetFocus((c.focus + 2) % 3)
	}
	switch c.focus {
	case cfMessage:
		return m.commitMessageKey(k)
	case cfButtons:
		switch key {
		case "esc":
			m.closeCommit()
		case "left", "h", "right", "l":
			c.button = 1 - c.button
		case "enter", " ":
			return m.doCommit(c.button == 1)
		case "k", "up":
			return m.commitSetFocus(cfMessage)
		}
		return nil
	}
	// Files.
	switch key {
	case "esc", "q":
		m.closeCommit()
		return nil
	case "a":
		m.toggleAllSelection()
		return nil
	case "enter":
		if n := m.selectedFileNode(); n != nil && n.isDir {
			m.fcollapsed[n.key] = !m.fcollapsed[n.key]
			m.rebuildFileTree()
			return nil
		}
		return m.commitSetFocus(cfMessage)
	case " ":
		if n := m.selectedFileNode(); n != nil {
			if n.isDir {
				m.toggleNodeSelection(n)
			} else {
				m.selected[n.file.Path] = !m.selected[n.file.Path]
			}
		}
		return nil
	case "left", "right":
		if n := m.selectedFileNode(); n != nil && n.isDir {
			m.fcollapsed[n.key] = key == "left"
			m.rebuildFileTree()
		}
		return nil
	case "d":
		if n := m.selectedFileNode(); n != nil && !n.isDir {
			return m.discardPrompt(*n.file)
		}
		return nil
	case "y":
		if n := m.selectedFileNode(); n != nil && !n.isDir {
			return copyToClipboard("path", n.file.Path)
		}
		return nil
	}
	filesRect, _, _ := m.commitLayout()
	prev := m.fcur
	if m.navigate(key, &m.fcur, len(m.fnodes), filesRect.h/2) && m.fcur != prev {
		return m.commitDiffLater()
	}
	return nil
}

func (m *Model) commitSetFocus(f commitFocus) tea.Cmd {
	c := m.commit
	c.focus = f
	if f == cfMessage {
		return c.msg.Focus()
	}
	c.msg.Blur()
	return nil
}

func (m *Model) commitMessageKey(k tea.KeyMsg) tea.Cmd {
	c := m.commit
	switch k.String() {
	case "esc":
		return m.commitSetFocus(cfFiles)
	case "up":
		if c.msg.Line() == 0 && len(c.history) > 0 {
			if c.histIdx == -1 {
				c.draft = c.msg.Value()
			}
			if c.histIdx+1 < len(c.history) {
				c.histIdx++
				c.msg.SetValue(c.history[c.histIdx])
			}
			return nil
		}
	case "down":
		if c.histIdx >= 0 && c.msg.Line() >= c.msg.LineCount()-1 {
			c.histIdx--
			if c.histIdx == -1 {
				c.msg.SetValue(c.draft)
			} else {
				c.msg.SetValue(c.history[c.histIdx])
			}
			return nil
		}
	}
	var cmd tea.Cmd
	c.msg, cmd = c.msg.Update(k)
	return cmd
}

// commitMouse handles clicks inside the commit workspace.
func (m *Model) commitMouse(msg tea.MouseMsg) tea.Cmd {
	filesRect, msgRect, btnRect := m.commitLayout()
	c := m.commit
	x, y := msg.X, msg.Y
	if msg.Button == tea.MouseButtonWheelUp || msg.Button == tea.MouseButtonWheelDown {
		delta := 3
		if msg.Button == tea.MouseButtonWheelUp {
			delta = -3
		}
		switch {
		case filesRect.contains(x, y):
			prev := m.fcur
			m.fcur = clamp(m.fcur+delta, 0, maxInt(0, len(m.fnodes)-1))
			if prev != m.fcur {
				return m.commitDiffLater()
			}
		case m.diffRect.contains(x, y) && m.diff != nil:
			m.diff.scroll = clamp(m.diff.scroll+delta, 0, m.diffMaxScroll())
		}
		return nil
	}
	if msg.Action != tea.MouseActionPress {
		return nil
	}
	switch {
	case filesRect.contains(x, y):
		row := m.fscroll + y - filesRect.y
		if row >= len(m.fnodes) {
			return m.commitSetFocus(cfFiles)
		}
		m.fcur = row
		cmd := m.commitSetFocus(cfFiles)
		n := &m.fnodes[row]
		rel := x - filesRect.x
		if msg.Button == tea.MouseButtonRight {
			x0, y0 := x, y
			m.menuAnchor = &[2]int{x0, y0}
			m.openMenu(m.menuForFile(n), x0, y0)
			m.menuAnchor = nil
			return cmd
		}
		if n.isDir {
			if rel <= 2+n.depth*2 {
				m.fcollapsed[n.key] = !m.fcollapsed[n.key]
				m.rebuildFileTree()
			} else {
				m.toggleNodeSelection(n)
			}
			return cmd
		}
		if rel <= 5+n.depth*2 {
			m.selected[n.file.Path] = !m.selected[n.file.Path]
		}
		return tea.Batch(cmd, m.commitDiffNow())
	case msgRect.contains(x, y):
		return m.commitSetFocus(cfMessage)
	case btnRect.contains(x, y):
		cmd := m.commitSetFocus(cfButtons)
		rel := x - btnRect.x
		commitW := width(theme.ButtonPrimary.Render("Commit"))
		if rel >= 1 && rel < 1+commitW {
			c.button = 0
			return tea.Batch(cmd, m.doCommit(false))
		}
		if rel >= 1+commitW+2 && rel < 1+commitW+2+width(theme.ButtonPrimary.Render("Commit & Push")) {
			c.button = 1
			return tea.Batch(cmd, m.doCommit(true))
		}
		return cmd
	}
	return nil
}

// ---- commit ------------------------------------------------------------

func (m *Model) doCommit(push bool) tea.Cmd {
	c := m.commit
	files := m.selectedFiles()
	if len(files) == 0 {
		return m.showToast("Select at least one file to commit", 2)
	}
	msg := strings.TrimSpace(c.msg.Value())
	if msg == "" {
		return tea.Batch(m.commitSetFocus(cfMessage), m.showToast("Enter a commit message", 2))
	}
	label := "Commit"
	if push {
		label = "Commit & Push"
	}
	repo := m.repo
	return m.actionThen(label, func() error { return repo.CommitPaths(msg, files) }, func(m *Model) tea.Cmd {
		c.msg.Reset()
		c.histIdx, c.draft = -1, ""
		c.history = append([]string{msg}, c.history...)
		m.closeCommit()
		if push {
			return m.openPush()
		}
		return nil
	})
}
