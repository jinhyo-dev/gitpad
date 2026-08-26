// Package ui implements the gitpad terminal interface.
package ui

import (
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/jinhyo-dev/gitpad/internal/git"
	"github.com/jinhyo-dev/gitpad/internal/graph"
	"github.com/jinhyo-dev/gitpad/internal/ui/theme"
)

// Panel identifies one of the three main panes.
type Panel int

const (
	PanelBranches Panel = iota
	PanelLog
	PanelChanges
	panelCount
)

type rect struct{ x, y, w, h int }

func (r rect) contains(x, y int) bool {
	return x >= r.x && x < r.x+r.w && y >= r.y && y < r.y+r.h
}

type searchKind int

const (
	searchText searchKind = iota
	searchAuthor
)

type toast struct {
	id   int
	text string
	kind int // 0 info, 1 ok, 2 err
}

// Model is the root bubbletea model.
type Model struct {
	repo  *git.Runner
	info  git.RepoInfo
	fatal error

	width, height int
	ready         bool // first WindowSizeMsg received
	loaded        bool // first data load finished
	focus         Panel

	// data
	refs         git.Refs
	commits      []git.Commit
	rows         []graph.Row
	graphW       int
	status       []git.FileChange
	hashIdx      map[string]int
	logOpts      git.LogOptions
	logTruncated bool
	fingerprint  string

	// branches panel
	bnodes  []bnode
	bexp    map[string]bool
	bcur    int
	bscroll int

	// log panel
	lcur    int
	lscroll int

	// changes panel
	files      []git.FileChange
	fnodes     []fnode
	fcollapsed map[string]bool
	fcur       int
	fscroll    int
	filesFor   string // hash or "local"
	details    *git.CommitDetails
	dscroll    int

	// diff view (replaces the log panel, or the right column in commit mode)
	diff     *diffState
	diffRect rect

	// commit workspace & push dialog
	selected   map[string]bool // commit checkboxes, keyed by path
	commit     *commitState
	commitOpen bool
	push       *pushState
	lastPull   string // merge | rebase | fetch

	// console tab
	console bool
	cscroll int

	// overlays
	menu   *menu
	dialog *dialog
	help   bool

	search     textinput.Model
	searching  bool
	searchKind searchKind

	spin     spinner.Model
	spinning bool // a spinner tick chain is alive
	selSeq   int  // debounce token for selection loads
	loading  int
	toast    *toast
	toastID  int
	seq      int
	dataSeq  int
	filesSeq int
	diffSeq  int

	rects        [panelCount]rect
	detailsRect  rect
	lastClickAt  time.Time
	lastClickKey string
	menuAnchor   *[2]int
}

// New creates the model for a repository at path.
func New(path string) Model {
	m := Model{
		focus:      PanelLog,
		bexp:       map[string]bool{"local": true, "remote": true},
		fcollapsed: map[string]bool{},
		logOpts:    git.LogOptions{All: true, Limit: 1000},
	}
	ti := textinput.New()
	ti.Prompt = ""
	ti.Placeholder = "Text or hash"
	ti.CharLimit = 200
	ti.Width = 30
	ti.PromptStyle = lipgloss.NewStyle()
	ti.TextStyle = lipgloss.NewStyle().Foreground(theme.Text)
	ti.PlaceholderStyle = lipgloss.NewStyle().Foreground(theme.Dim)
	ti.Cursor.Style = lipgloss.NewStyle().Foreground(theme.Accent)
	m.search = ti

	sp := spinner.New()
	sp.Spinner = spinner.MiniDot
	sp.Style = lipgloss.NewStyle().Foreground(theme.Accent)
	m.spin = sp

	repo, err := git.Open(path)
	if err != nil {
		m.fatal = err
		return m
	}
	m.repo = repo
	return m
}

// Init kicks off the first load. The load itself is started from Update (on
// initMsg) so that request sequencing happens on the live model, not a copy.
func (m Model) Init() tea.Cmd {
	if m.fatal != nil {
		return nil
	}
	return func() tea.Msg { return initMsg{} }
}

// ---- log row helpers ---------------------------------------------------

func (m *Model) hasLocalRow() bool {
	return len(m.status) > 0 && m.logOpts.Ref == "" && m.logOpts.Grep == "" && m.logOpts.Author == "" && len(m.logOpts.Paths) == 0
}

func (m *Model) logLen() int {
	n := len(m.commits)
	if m.hasLocalRow() {
		n++
	}
	return n
}

// commitAt maps a log row index to a commit (nil for the local-changes row).
func (m *Model) commitAt(row int) *git.Commit {
	if m.hasLocalRow() {
		row--
	}
	if row < 0 || row >= len(m.commits) {
		return nil
	}
	return &m.commits[row]
}

func (m *Model) rowOfHash(hash string) int {
	i, ok := m.hashIdx[hash]
	if !ok {
		return -1
	}
	if m.hasLocalRow() {
		i++
	}
	return i
}

func (m *Model) selectedCommit() *git.Commit { return m.commitAt(m.lcur) }

func (m *Model) isLocalRow(row int) bool { return m.hasLocalRow() && row == 0 }

func (m *Model) currentBranch() string {
	if m.info.Detached {
		return "HEAD"
	}
	return m.info.Head
}

func (m *Model) selectedBranchNode() *bnode {
	if m.bcur < 0 || m.bcur >= len(m.bnodes) {
		return nil
	}
	return &m.bnodes[m.bcur]
}

func (m *Model) selectedFileNode() *fnode {
	if m.fcur < 0 || m.fcur >= len(m.fnodes) {
		return nil
	}
	return &m.fnodes[m.fcur]
}
