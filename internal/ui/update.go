package ui

import (
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/jinhyo/gitpad/internal/git"
	"github.com/jinhyo/gitpad/internal/graph"
	"github.com/jinhyo/gitpad/internal/ui/theme"
)

// Update handles messages.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	mm := &m
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		mm.width, mm.height = msg.Width, msg.Height
		mm.ready = true
		mm.layout()
	case initMsg:
		cmd = tea.Batch(mm.loadAll(""), watchTick())
	case tea.KeyMsg:
		if mm.fatal != nil {
			return m, tea.Quit
		}
		cmd = mm.handleKey(msg)
	case tea.MouseMsg:
		cmd = mm.handleMouse(msg)
	case spinner.TickMsg:
		if mm.loading > 0 {
			mm.spin, cmd = mm.spin.Update(msg)
		} else {
			mm.spinning = false // chain ends; restarted below when needed
		}
	case selectMsg:
		if msg.seq == mm.selSeq {
			cmd = mm.refreshSelection(false)
		}
	case dataMsg:
		cmd = mm.onData(msg)
	case filesMsg:
		cmd = mm.onFiles(msg)
	case diffMsg:
		mm.loading--
		if msg.seq == mm.diffSeq && mm.diff != nil {
			mm.diff.loading = false
			mm.diff.err = msg.err
			mm.diff.lines, mm.diff.stats = parseDiff(msg.text)
			mm.diff.scroll = clamp(mm.diff.scroll, 0, mm.diffMaxScroll())
		}
	case actionDoneMsg:
		mm.loading--
		if mm.push != nil {
			mm.push.busy = false
		}
		if msg.err != nil {
			if msg.onErr != nil {
				cmd = msg.onErr(mm, msg.err)
			} else {
				cmd = mm.showToast(msg.label+": "+msg.err.Error(), 2)
			}
			return m, tea.Batch(cmd, mm.loadAll(msg.keepHash))
		}
		cmd = tea.Batch(mm.showToast(msg.label+" ✓", 1), mm.loadAll(msg.keepHash))
		if msg.then != nil {
			cmd = tea.Batch(cmd, msg.then(mm))
		}
	case historyMsg:
		if mm.commit != nil {
			mm.commit.history = msg.msgs
		}
	case commitDiffMsg:
		if mm.commitOpen && mm.commit != nil && msg.seq == mm.commit.diffSeq {
			cmd = mm.commitDiffNow()
		}
	case pushDataMsg:
		mm.onPushData(msg)
	case toastClearMsg:
		if mm.toast != nil && mm.toast.id == msg.id {
			mm.toast = nil
		}
	case watchTickMsg:
		if mm.loading == 0 && mm.fatal == nil {
			return m, tea.Batch(mm.fingerprintCmd(), watchTick())
		}
		return m, watchTick()
	case watchResultMsg:
		if msg.fp != mm.fingerprint && mm.loading == 0 && mm.dialog == nil && mm.menu == nil {
			keep := ""
			if c := mm.selectedCommit(); c != nil {
				keep = c.Hash
			}
			cmd = mm.loadAll(keep)
		}
	case clipboardMsg:
		if msg.err != nil {
			cmd = mm.showToast("copy failed: "+msg.err.Error(), 2)
		} else {
			cmd = mm.showToast("copied "+msg.what, 1)
		}
	}
	if mm.loading < 0 {
		mm.loading = 0
	}
	// Start exactly one spinner tick chain per loading period. Batching a
	// tick on every message would create an immediate-message busy loop.
	if mm.loading > 0 && !mm.spinning {
		mm.spinning = true
		cmd = tea.Batch(cmd, mm.spin.Tick)
	}
	return m, cmd
}

// scheduleSelection loads files for the cursor row after a short debounce so
// holding j/k doesn't spawn a git process per step.
func (m *Model) scheduleSelection() tea.Cmd {
	key := "local"
	if c := m.selectedCommit(); c != nil {
		key = c.Hash
	}
	if key == m.filesFor {
		return nil
	}
	m.selSeq++
	seq := m.selSeq
	return tea.Tick(selectDebounce, func(time.Time) tea.Msg { return selectMsg{seq: seq} })
}

func (m *Model) onData(msg dataMsg) tea.Cmd {
	m.loading--
	if msg.seq != m.dataSeq {
		return nil
	}
	m.loaded = true
	if msg.err != nil {
		return m.showToast(msg.err.Error(), 2)
	}
	m.info = msg.info
	m.refs = msg.refs
	m.commits = msg.commits
	m.status = msg.status
	m.fingerprint = msg.fp
	m.syncSelection()
	if m.commitOpen && len(m.status) == 0 {
		m.closeCommit()
	}
	m.logTruncated = len(msg.commits) >= m.logOpts.Limit

	m.hashIdx = make(map[string]int, len(m.commits))
	nodes := make([]graph.Node, len(m.commits))
	for i, c := range m.commits {
		m.hashIdx[c.Hash] = i
		nodes[i] = graph.Node{Hash: c.Hash, Parents: c.Parents}
	}
	m.rows = graph.Build(nodes, len(theme.Lanes))
	m.graphW = 1
	for _, r := range m.rows {
		m.graphW = maxInt(m.graphW, len(r.Cells))
	}
	m.graphW = minInt(m.graphW, maxGraphCells)
	m.rebuildBranchTree()

	if msg.keepHash != "" {
		if row := m.rowOfHash(msg.keepHash); row >= 0 {
			m.lcur = row
		}
	}
	m.lcur = clamp(m.lcur, 0, maxInt(0, m.logLen()-1))
	return m.refreshSelection(true)
}

func (m *Model) onFiles(msg filesMsg) tea.Cmd {
	m.loading--
	if msg.seq != m.filesSeq {
		return nil
	}
	if msg.err != nil {
		return m.showToast(msg.err.Error(), 2)
	}
	changed := msg.key != m.filesFor
	m.filesFor = msg.key
	m.files = msg.files
	m.details = msg.details
	if changed {
		m.fcur, m.fscroll, m.dscroll = 0, 0, 0
		m.fcollapsed = map[string]bool{}
	}
	m.rebuildFileTree()
	if m.commitOpen && msg.key == "local" {
		return m.commitDiffNow()
	}
	// Keep an open diff in sync with the refreshed file list.
	if m.diff != nil {
		if changed {
			m.diff = nil
		} else {
			for i := range m.files {
				if m.files[i].Path == m.diff.path {
					return m.openDiff(m.files[i])
				}
			}
			m.diff = nil
		}
	}
	return nil
}

// refreshSelection loads files for the commit under the cursor.
func (m *Model) refreshSelection(force bool) tea.Cmd {
	key := "local"
	c := m.selectedCommit()
	if c != nil {
		key = c.Hash
	} else if !m.isLocalRow(m.lcur) {
		m.files, m.fnodes, m.details, m.filesFor = nil, nil, nil, ""
		return nil
	}
	if !force && key == m.filesFor {
		return nil
	}
	return m.loadFiles(c)
}

// ---- keyboard ----------------------------------------------------------

func (m *Model) handleKey(k tea.KeyMsg) tea.Cmd {
	key := k.String()
	if key == "ctrl+c" {
		return tea.Quit
	}
	switch {
	case m.dialog != nil:
		return m.dialogKey(k)
	case m.menu != nil:
		return m.menuKey(k)
	case m.help:
		if key == "esc" || key == "?" || key == "q" {
			m.help = false
		}
		return nil
	case m.searching:
		return m.searchKey(k)
	case m.push != nil:
		return m.pushKey(k)
	case m.commitOpen:
		return m.commitKey(k)
	}

	// Global keys.
	switch key {
	case "q":
		if m.diff != nil {
			m.diff = nil
			return nil
		}
		if m.console {
			m.console = false
			return nil
		}
		return tea.Quit
	case "?":
		m.help = true
		return nil
	case "tab":
		m.focus = (m.focus + 1) % panelCount
		return nil
	case "shift+tab":
		m.focus = (m.focus + panelCount - 1) % panelCount
		return nil
	case "1":
		m.focus = PanelBranches
		return nil
	case "2":
		m.focus = PanelLog
		return nil
	case "3":
		m.focus = PanelChanges
		return nil
	case "h":
		if m.focus > 0 {
			m.focus--
		}
		return nil
	case "l":
		if m.focus < panelCount-1 {
			m.focus++
		}
		return nil
	case "r":
		return m.reload()
	case "f":
		return m.action("Fetch", m.repo.Fetch)
	case "p":
		if m.diff != nil {
			return m.diffStep(-1)
		}
		return m.pullChooser()
	case "P":
		return m.openPush()
	case "C":
		return m.openCommit()
	case "/":
		return m.startSearch(searchText)
	case "a":
		if m.focus == PanelChanges && m.filesFor == "local" {
			m.toggleAllSelection()
			return nil
		}
		return m.startSearch(searchAuthor)
	case "A":
		m.logOpts.All = !m.logOpts.All
		m.logOpts.Ref = ""
		return m.applyFilter()
	case "`":
		m.console = !m.console
		m.cscroll = 0
		if m.console {
			m.focus = PanelLog
		}
		return nil
	case "esc":
		switch {
		case m.diff != nil:
			m.diff = nil
		case m.console:
			m.console = false
		case m.hasFilter():
			m.logOpts = git.LogOptions{All: true, Limit: 1000}
			return m.applyFilter()
		}
		return nil
	}

	if m.diff != nil && m.focus != PanelBranches {
		if cmd, handled := m.diffKey(key); handled {
			return cmd
		}
	}
	if m.console && m.focus == PanelLog {
		switch key {
		case "j", "down":
			m.cscroll = maxInt(0, m.cscroll-1)
		case "k", "up":
			m.cscroll++
		case "G":
			m.cscroll = 0
		}
		return nil
	}
	switch m.focus {
	case PanelBranches:
		return m.branchesKey(key)
	case PanelLog:
		return m.logKey(key)
	default:
		return m.filesKey(key)
	}
}

func (m *Model) navigate(key string, cur *int, n, page int) bool {
	if n == 0 {
		return false
	}
	switch key {
	case "j", "down":
		*cur = minInt(*cur+1, n-1)
	case "k", "up":
		*cur = maxInt(*cur-1, 0)
	case "g", "home":
		*cur = 0
	case "G", "end":
		*cur = n - 1
	case "ctrl+d", "pgdown":
		*cur = minInt(*cur+page, n-1)
	case "ctrl+u", "pgup":
		*cur = maxInt(*cur-page, 0)
	default:
		return false
	}
	return true
}

func (m *Model) branchesKey(key string) tea.Cmd {
	if m.navigate(key, &m.bcur, len(m.bnodes), m.rects[PanelBranches].h/2) {
		return nil
	}
	n := m.selectedBranchNode()
	if n == nil {
		return nil
	}
	switch key {
	case "enter", "m":
		if n.kind == bSection || n.kind == bFolder {
			if key == "enter" {
				m.bexp[n.key] = !m.bexp[n.key]
				m.rebuildBranchTree()
				return nil
			}
			return nil
		}
		x, y := m.menuAnchorFor(PanelBranches, m.bcur)
		m.openMenu(m.menuForBranch(n), x, y)
	case " ":
		if n.kind == bSection || n.kind == bFolder {
			m.bexp[n.key] = !m.bexp[n.key]
			m.rebuildBranchTree()
		}
	case "right":
		// Unfold if possible, otherwise move on to the next section.
		if (n.kind == bSection || n.kind == bFolder) && !m.bexp[n.key] {
			m.bexp[n.key] = true
			m.rebuildBranchTree()
			return nil
		}
		m.focus = PanelLog
	case "left":
		switch {
		case (n.kind == bSection || n.kind == bFolder) && m.bexp[n.key]:
			m.bexp[n.key] = false
			m.rebuildBranchTree()
		case n.kind == bLeaf || n.kind == bFolder:
			// jump to the parent folder
			for i := m.bcur - 1; i >= 0; i-- {
				if m.bnodes[i].kind != bLeaf && m.bnodes[i].depth < n.depth {
					m.bcur = i
					return nil
				}
			}
			m.focus = PanelChanges // leftmost pane: wrap around
		default:
			m.focus = PanelChanges
		}
	case "c":
		if n.branch != nil {
			return m.checkoutBranch(n.branch)
		}
	case "s":
		if n.branch != nil {
			return m.showRefInLog(n.branch.Name)
		}
		if n.kind == bHead {
			return m.showRefInLog("")
		}
	case "d":
		if n.branch != nil {
			return m.deleteRef(n.branch)
		}
	}
	return nil
}

func (m *Model) logKey(key string) tea.Cmd {
	prev := m.lcur
	if m.navigate(key, &m.lcur, m.logLen(), m.rects[PanelLog].h/2) {
		var cmd tea.Cmd
		if m.lcur != prev {
			cmd = m.scheduleSelection()
		}
		if m.lcur >= m.logLen()-1 && m.logTruncated {
			cmd = tea.Batch(cmd, m.loadMore())
		}
		return cmd
	}
	switch key {
	case "left":
		m.focus = PanelBranches
		return nil
	case "right":
		m.focus = PanelChanges
		return nil
	case "enter", "m":
		x, y := m.menuAnchorFor(PanelLog, m.lcur)
		if m.isLocalRow(m.lcur) {
			m.openMenu(m.menuForLocal(), x, y)
		} else if c := m.selectedCommit(); c != nil {
			m.openMenu(m.menuForCommit(c), x, y)
		}
	case "y":
		if c := m.selectedCommit(); c != nil {
			return copyToClipboard("hash "+c.Short, c.Hash)
		}
	case "c":
		return m.openCommit()
	case "d", " ":
		// Open the diff of the first file of the selection.
		for _, n := range m.fnodes {
			if !n.isDir {
				m.focus = PanelChanges
				return m.openDiff(*n.file)
			}
		}
	}
	return nil
}

func (m *Model) filesKey(key string) tea.Cmd {
	if m.navigate(key, &m.fcur, len(m.fnodes), m.rects[PanelChanges].h/2) {
		if m.diff != nil {
			if n := m.selectedFileNode(); n != nil && !n.isDir {
				return m.openDiff(*n.file)
			}
		}
		return nil
	}
	n := m.selectedFileNode()
	switch key {
	case "J":
		m.dscroll++
		return nil
	case "K":
		m.dscroll = maxInt(0, m.dscroll-1)
		return nil
	}
	if n == nil {
		return nil
	}
	switch key {
	case "enter":
		if n.isDir {
			m.fcollapsed[n.key] = !m.fcollapsed[n.key]
			m.rebuildFileTree()
			return nil
		}
		if m.diff != nil && m.diff.path == n.file.Path {
			m.diff = nil
			return nil
		}
		return m.openDiff(*n.file)
	case " ":
		if m.filesFor == "local" {
			if n.isDir && !n.isGroup {
				m.fcollapsed[n.key] = !m.fcollapsed[n.key]
				m.rebuildFileTree()
			} else {
				m.toggleNodeSelection(n)
			}
			return nil
		}
		if n.isDir {
			m.fcollapsed[n.key] = !m.fcollapsed[n.key]
			m.rebuildFileTree()
		}
	case "right":
		if n.isDir && m.fcollapsed[n.key] {
			m.fcollapsed[n.key] = false
			m.rebuildFileTree()
			return nil
		}
		m.focus = PanelBranches // rightmost pane: wrap around
	case "left":
		if n.isDir && !m.fcollapsed[n.key] {
			m.fcollapsed[n.key] = true
			m.rebuildFileTree()
			return nil
		}
		m.focus = PanelLog
	case "m":
		x, y := m.menuAnchorFor(PanelChanges, m.fcur)
		m.openMenu(m.menuForFile(n), x, y)
	case "a":
		if m.filesFor == "local" {
			m.toggleAllSelection()
		}
	case "y":
		if !n.isDir {
			return copyToClipboard("path", n.file.Path)
		}
	case "H":
		if !n.isDir {
			return m.showPathHistory(n.file.Path)
		}
	case "s":
		if !n.isDir && m.filesFor == "local" {
			return m.toggleStage(*n.file)
		}
	case "c":
		return m.openCommit()
	case "d":
		if !n.isDir && m.filesFor == "local" {
			return m.discardPrompt(*n.file)
		}
	}
	return nil
}

func (m *Model) diffKey(key string) (tea.Cmd, bool) {
	d := m.diff
	h := m.rects[PanelLog].h
	switch key {
	case "j", "down":
		d.scroll = minInt(d.scroll+1, m.diffMaxScroll())
	case "k", "up":
		d.scroll = maxInt(d.scroll-1, 0)
	case "ctrl+d", "pgdown", " ":
		d.scroll = minInt(d.scroll+h/2, m.diffMaxScroll())
	case "ctrl+u", "pgup":
		d.scroll = maxInt(d.scroll-h/2, 0)
	case "g", "home":
		d.scroll = 0
	case "G", "end":
		d.scroll = m.diffMaxScroll()
	case "n":
		return m.diffStep(1), true
	case "p":
		return m.diffStep(-1), true
	default:
		return nil, false
	}
	return nil, true
}

// diffStep moves to the next/previous file while the diff is open.
func (m *Model) diffStep(dir int) tea.Cmd {
	for i := m.fcur + dir; i >= 0 && i < len(m.fnodes); i += dir {
		if !m.fnodes[i].isDir {
			m.fcur = i
			return m.openDiff(*m.fnodes[i].file)
		}
	}
	return nil
}

func (m *Model) openDiff(fc git.FileChange) tea.Cmd {
	var c *git.Commit
	if m.filesFor != "local" {
		c = m.selectedCommit()
	}
	m.console = false
	m.diff = &diffState{path: fc.Path, commit: c, file: fc, loading: true}
	return m.loadDiff(c, fc)
}

func (m *Model) menuKey(k tea.KeyMsg) tea.Cmd {
	mn := m.menu
	switch k.String() {
	case "esc", "q":
		if mn.parent != nil {
			m.menu = mn.parent
		} else {
			m.menu = nil
		}
	case "left":
		if mn.parent != nil {
			m.menu = mn.parent
		}
	case "j", "down":
		mn.move(1)
	case "k", "up":
		mn.move(-1)
	case "enter", "right", " ":
		return m.activateMenuItem()
	default:
		s := k.String()
		for i, it := range mn.items {
			if !it.sep && !it.disabled && it.key != "" && it.key == s {
				mn.cur = i
				return m.activateMenuItem()
			}
		}
	}
	return nil
}

func (m *Model) dialogKey(k tea.KeyMsg) tea.Cmd {
	d := m.dialog
	switch k.String() {
	case "esc":
		m.dialog = nil
		return nil
	case "enter":
		value := ""
		if d.kind == dlgInput {
			value = strings.TrimSpace(d.input.Value())
			if value == "" {
				return nil
			}
		}
		m.dialog = nil
		if d.onOK != nil {
			return d.onOK(m, value)
		}
		return nil
	}
	if d.kind == dlgConfirm {
		switch k.String() {
		case "y", "Y":
			m.dialog = nil
			return d.onOK(m, "")
		case "n", "N":
			m.dialog = nil
		}
		return nil
	}
	var cmd tea.Cmd
	d.input, cmd = d.input.Update(k)
	return cmd
}

func (m *Model) startSearch(kind searchKind) tea.Cmd {
	m.searching = true
	m.searchKind = kind
	if kind == searchAuthor {
		m.search.Placeholder = "author name or email"
		m.search.SetValue(m.logOpts.Author)
	} else {
		m.search.Placeholder = "Text or hash"
		m.search.SetValue(m.logOpts.Grep)
	}
	m.search.CursorEnd()
	return m.search.Focus()
}

func (m *Model) searchKey(k tea.KeyMsg) tea.Cmd {
	switch k.String() {
	case "esc":
		m.searching = false
		m.search.Blur()
		return nil
	case "enter":
		m.searching = false
		m.search.Blur()
		v := strings.TrimSpace(m.search.Value())
		if m.searchKind == searchAuthor {
			m.logOpts.Author = v
			return m.applyFilter()
		}
		// A hash prefix jumps straight to the commit.
		if isHexPrefix(v) {
			if full, ok := m.repo.ResolveHash(v); ok {
				if row := m.rowOfHash(full); row >= 0 {
					m.lcur = row
					m.focus = PanelLog
					return m.refreshSelection(false)
				}
				m.logOpts.Grep = ""
				m.logOpts.Ref = full
				return m.applyFilter()
			}
		}
		m.logOpts.Grep = v
		return m.applyFilter()
	}
	var cmd tea.Cmd
	m.search, cmd = m.search.Update(k)
	return cmd
}

func isHexPrefix(s string) bool {
	if len(s) < 5 || len(s) > 40 {
		return false
	}
	for _, r := range s {
		if !strings.ContainsRune("0123456789abcdefABCDEF", r) {
			return false
		}
	}
	return true
}

// ---- mouse ---------------------------------------------------------------

func (m *Model) panelAt(x, y int) (Panel, bool) {
	for p := Panel(0); p < panelCount; p++ {
		if m.rects[p].contains(x, y) {
			return p, true
		}
	}
	return 0, false
}

func (m *Model) handleMouse(msg tea.MouseMsg) tea.Cmd {
	if m.dialog != nil || m.help {
		if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft && m.help {
			m.help = false
		}
		return nil
	}
	if m.push != nil {
		return m.pushMouse(msg)
	}
	if m.menu != nil {
		switch {
		case msg.Button == tea.MouseButtonWheelUp:
			m.menu.move(-1)
		case msg.Button == tea.MouseButtonWheelDown:
			m.menu.move(1)
		case msg.Action == tea.MouseActionMotion:
			if i := m.menu.itemAt(msg.X, msg.Y); i >= 0 {
				m.menu.cur = i
			}
		case msg.Action == tea.MouseActionPress:
			if i := m.menu.itemAt(msg.X, msg.Y); i >= 0 {
				m.menu.cur = i
				return m.activateMenuItem()
			}
			if !m.menu.inside(msg.X, msg.Y) {
				m.menu = nil
			}
		}
		return nil
	}

	if m.commitOpen && (m.rects[PanelLog].contains(msg.X, msg.Y) || m.diffRect.contains(msg.X, msg.Y)) {
		return m.commitMouse(msg)
	}
	switch msg.Button {
	case tea.MouseButtonWheelUp, tea.MouseButtonWheelDown:
		delta := 3
		if msg.Button == tea.MouseButtonWheelUp {
			delta = -3
		}
		return m.scrollAt(msg.X, msg.Y, delta)
	}
	if m.commitOpen && (m.rects[PanelLog].contains(msg.X, msg.Y) || m.diffRect.contains(msg.X, msg.Y)) {
		return m.commitMouse(msg)
	}
	if msg.Action != tea.MouseActionPress {
		return nil
	}
	if m.searching {
		m.searching = false
		m.search.Blur()
	}
	switch msg.Y {
	case rowHeader:
		return m.headerClick(msg.X)
	case rowFilter:
		return m.filterClick(msg.X)
	}
	p, ok := m.panelAt(msg.X, msg.Y)
	if !ok {
		if m.detailsRect.contains(msg.X, msg.Y) {
			m.focus = PanelChanges
		}
		return nil
	}
	m.focus = p
	row := m.scrollOf(p) + msg.Y - m.rects[p].y
	right := msg.Button == tea.MouseButtonRight
	switch p {
	case PanelBranches:
		if row >= len(m.bnodes) {
			return nil
		}
		m.bcur = row
		if right {
			return m.branchesKey("m")
		}
		if m.doubleClick("b", row) {
			return m.branchesKey("enter")
		}
		if n := m.selectedBranchNode(); n != nil && (n.kind == bSection || n.kind == bFolder) {
			// Single click on the arrow area toggles the folder.
			if msg.X-m.rects[p].x <= 2+n.depth*2 {
				return m.branchesKey(" ")
			}
		}
	case PanelLog:
		if m.console || m.diff != nil {
			return nil
		}
		if row >= m.logLen() {
			return nil
		}
		prev := m.lcur
		m.lcur = row
		var cmd tea.Cmd
		if prev != row {
			cmd = m.refreshSelection(false)
		}
		if right {
			m.menuAnchor = &[2]int{msg.X, msg.Y}
			c := m.logKey("m")
			m.menuAnchor = nil
			return tea.Batch(cmd, c)
		}
		if m.doubleClick("l", row) {
			return tea.Batch(cmd, m.logKey("enter"))
		}
		return cmd
	case PanelChanges:
		if row >= len(m.fnodes) {
			return nil
		}
		m.fcur = row
		if right {
			return m.filesKey("m")
		}
		if n := m.selectedFileNode(); n != nil {
			rel := msg.X - m.rects[p].x
			if n.isDir && rel <= 2+n.depth*2 {
				m.fcollapsed[n.key] = !m.fcollapsed[n.key]
				m.rebuildFileTree()
				return nil
			}
			if m.filesFor == "local" && !n.isDir && rel <= 5+n.depth*2 {
				m.selected[n.file.Path] = !m.selected[n.file.Path]
				return nil
			}
		}
		if m.doubleClick("f", row) {
			return m.filesKey("enter")
		}
		if m.diff != nil {
			if n := m.selectedFileNode(); n != nil && !n.isDir {
				return m.openDiff(*n.file)
			}
		}
	}
	return nil
}

func (m *Model) doubleClick(kind string, row int) bool {
	key := kind + ":" + itoa(row)
	now := time.Now()
	hit := m.lastClickKey == key && now.Sub(m.lastClickAt) < 400*time.Millisecond
	m.lastClickAt, m.lastClickKey = now, key
	if hit {
		m.lastClickKey = ""
	}
	return hit
}

func itoa(i int) string { return strconv.Itoa(i) }

func (m *Model) scrollAt(x, y, delta int) tea.Cmd {
	if m.detailsRect.contains(x, y) {
		m.dscroll = maxInt(0, m.dscroll+delta)
		return nil
	}
	p, ok := m.panelAt(x, y)
	if !ok {
		return nil
	}
	switch p {
	case PanelBranches:
		m.bcur = clamp(m.bcur+delta, 0, maxInt(0, len(m.bnodes)-1))
	case PanelLog:
		switch {
		case m.diff != nil:
			m.diff.scroll = clamp(m.diff.scroll+delta, 0, m.diffMaxScroll())
		case m.console:
			m.cscroll = maxInt(0, m.cscroll-delta)
		default:
			prev := m.lcur
			m.lcur = clamp(m.lcur+delta, 0, maxInt(0, m.logLen()-1))
			var cmd tea.Cmd
			if prev != m.lcur {
				cmd = m.scheduleSelection()
			}
			if m.lcur >= m.logLen()-1 && m.logTruncated {
				cmd = tea.Batch(cmd, m.loadMore())
			}
			return cmd
		}
	case PanelChanges:
		m.fcur = clamp(m.fcur+delta, 0, maxInt(0, len(m.fnodes)-1))
	}
	return nil
}

func (m *Model) headerClick(x int) tea.Cmd {
	// Tabs live at the right edge: "[ Log ][ Console ]  ? help "
	tail := width(theme.TabActive.Render("Log")) + width(theme.TabInactive.Render("Console")) + 2 + width(theme.KeyHint.Render("?")+theme.KeyLabel.Render(" help "))
	start := m.width - tail
	logW := width(theme.TabActive.Render("Log"))
	conW := width(theme.TabInactive.Render("Console"))
	switch {
	case x >= start && x < start+logW:
		m.console = false
	case x >= start+logW && x < start+logW+conW:
		m.console = true
		m.focus = PanelLog
	case x >= start+logW+conW:
		m.help = true
	}
	return nil
}

func (m *Model) filterClick(x int) tea.Cmd {
	// Reconstruct chip extents in the same order as renderFilterBar.
	pos := 1
	searchW := width(theme.FilterChip.Render("⌕ " + trunc(orDefault(m.logOpts.Grep, "Text or hash"), 28)))
	if x >= pos && x < pos+searchW {
		return m.startSearch(searchText)
	}
	pos += searchW + 2
	branchLabel := "Branch: All"
	if m.logOpts.Ref != "" {
		branchLabel = "Branch: " + trunc(m.logOpts.Ref, 24)
	} else if !m.logOpts.All {
		branchLabel = "Branch: " + trunc(m.currentBranch(), 24)
	}
	bw := width(theme.FilterChip.Render(branchLabel))
	if x >= pos && x < pos+bw {
		m.logOpts.All = !m.logOpts.All
		m.logOpts.Ref = ""
		return m.applyFilter()
	}
	pos += bw + 2
	userLabel := "User: Any"
	if m.logOpts.Author != "" {
		userLabel = "User: " + trunc(m.logOpts.Author, 20)
	}
	uw := width(theme.FilterChip.Render(userLabel))
	if x >= pos && x < pos+uw {
		return m.startSearch(searchAuthor)
	}
	pos += uw + 2
	if len(m.logOpts.Paths) > 0 {
		pw := width(theme.FilterChip.Render("Path: " + trunc(m.logOpts.Paths[0], 30)))
		if x >= pos && x < pos+pw {
			m.logOpts.Paths = nil
			return m.applyFilter()
		}
	}
	return nil
}

func orDefault(s, d string) string {
	if s == "" {
		return d
	}
	return s
}
