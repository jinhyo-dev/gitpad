package ui

import (
	"context"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/jinhyo-dev/gitpad/internal/ci"
	"github.com/jinhyo-dev/gitpad/internal/git"
)

// Timings for the CI column.
var (
	ciRefreshInterval = 30 * time.Second
	ciLookahead       = 60 // rows below the viewport to prefetch
	ciLookbehind      = 20
)

// initCI detects the hosting provider from the origin remote.
func (m *Model) initCI() tea.Cmd {
	repo := m.repo
	return func() tea.Msg {
		out, err := repo.Run("remote", "get-url", "origin")
		if err != nil {
			return ciInitMsg{}
		}
		return ciInitMsg{p: ci.Detect(strings.TrimSpace(out))}
	}
}

// fetchCI requests status for commits around the viewport that are not
// cached yet. One request is in flight at a time.
func (m *Model) fetchCI() tea.Cmd {
	if m.ci == nil || m.ciBusy || m.ciErr || m.commitOpen {
		return nil
	}
	h := m.rects[PanelLog].h
	start := maxInt(0, m.lscroll-ciLookbehind)
	end := minInt(m.logLen(), m.lscroll+h+ciLookahead)
	var shas []string
	for row := start; row < end; row++ {
		c := m.commitAt(row)
		if c == nil {
			continue
		}
		if _, ok := m.ciResults[c.Hash]; ok || m.ciPending[c.Hash] {
			continue
		}
		shas = append(shas, c.Hash)
	}
	if len(shas) == 0 {
		return nil
	}
	for _, s := range shas {
		m.ciPending[s] = true
	}
	m.ciBusy = true
	p := m.ci
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		defer cancel()
		res, err := p.Fetch(ctx, shas)
		return ciMsg{shas: shas, results: res, err: err}
	}
}

func ciRefreshTick() tea.Cmd {
	return tea.Tick(ciRefreshInterval, func(time.Time) tea.Msg { return ciRefreshMsg{} })
}

type dataMsg struct {
	seq      int
	info     git.RepoInfo
	refs     git.Refs
	commits  []git.Commit
	status   []git.FileChange
	err      error
	keepHash string
	fp       string
}

type filesMsg struct {
	seq     int
	key     string
	files   []git.FileChange
	details *git.CommitDetails
	err     error
}

type diffMsg struct {
	seq  int
	path string
	text string
	err  error
}

type actionDoneMsg struct {
	label    string
	err      error
	keepHash string
	onErr    func(m *Model, err error) tea.Cmd
	then     func(m *Model) tea.Cmd // runs on success, before the reload
}

type initMsg struct{}
type ciInitMsg struct{ p ci.Provider }
type ciMsg struct {
	shas    []string
	results map[string]ci.Result
	err     error
}
type ciRefreshMsg struct{}
type selectMsg struct{ seq int }
type toastClearMsg struct{ id int }
type watchTickMsg struct{}
type watchResultMsg struct{ fp string }
type clipboardMsg struct {
	err  error
	what string
}

// Timings are variables so tests can shorten them.
var (
	watchInterval    = 3 * time.Second
	selectDebounce   = 60 * time.Millisecond
	toastDuration    = 3 * time.Second
	toastErrDuration = 6 * time.Second
)

func watchTick() tea.Cmd {
	return tea.Tick(watchInterval, func(time.Time) tea.Msg { return watchTickMsg{} })
}

func (m *Model) fingerprintCmd() tea.Cmd {
	repo := m.repo
	return func() tea.Msg { return watchResultMsg{fp: computeFingerprint(repo)} }
}

func computeFingerprint(repo *git.Runner) string {
	h := sha1.New()
	out, _ := repo.Run("rev-parse", "HEAD")
	h.Write([]byte(out))
	out, _ = repo.Run("for-each-ref", "--format=%(refname)%(objectname)")
	h.Write([]byte(out))
	out, _ = repo.Run("status", "--porcelain=v1", "-z", "--untracked-files=all")
	h.Write([]byte(out))
	return fmt.Sprintf("%x", h.Sum(nil))
}

// loadAll reloads refs, log, status and repo info.
func (m *Model) loadAll(keepHash string) tea.Cmd {
	m.loading++
	m.seq++
	seq := m.seq
	m.dataSeq = seq
	repo := m.repo
	opts := m.logOpts
	return func() tea.Msg {
		msg := dataMsg{seq: seq, keepHash: keepHash}
		msg.info = repo.Info()
		refs, err := repo.Refs()
		if err != nil {
			msg.err = err
			return msg
		}
		msg.refs = refs
		commits, err := repo.Log(opts)
		if err != nil {
			msg.err = err
			return msg
		}
		msg.commits = commits
		msg.status, _ = repo.Status()
		msg.fp = computeFingerprint(repo)
		return msg
	}
}

// loadFiles fetches the file list + details for a commit, or the working tree.
func (m *Model) loadFiles(c *git.Commit) tea.Cmd {
	m.loading++
	m.seq++
	seq := m.seq
	m.filesSeq = seq
	repo := m.repo
	if c == nil {
		status := m.status
		return func() tea.Msg {
			return filesMsg{seq: seq, key: "local", files: status}
		}
	}
	commit := *c
	return func() tea.Msg {
		msg := filesMsg{seq: seq, key: commit.Hash}
		files, err := repo.CommitFiles(commit)
		if err != nil {
			msg.err = err
			return msg
		}
		msg.files = files
		msg.details, msg.err = repo.Details(commit.Hash)
		return msg
	}
}

func (m *Model) loadDiff(c *git.Commit, fc git.FileChange) tea.Cmd {
	m.loading++
	m.seq++
	seq := m.seq
	m.diffSeq = seq
	repo := m.repo
	var commit *git.Commit
	if c != nil {
		cc := *c
		commit = &cc
	}
	return func() tea.Msg {
		var text string
		var err error
		if commit == nil {
			text, err = repo.WorktreeDiff(fc)
		} else {
			text, err = repo.CommitDiff(*commit, fc.Path)
		}
		return diffMsg{seq: seq, path: fc.Path, text: text, err: err}
	}
}

// action runs a mutating git operation asynchronously.
func (m *Model) action(label string, fn func() error) tea.Cmd {
	return m.actionWith(label, fn, nil)
}

// actionWith is action with a custom error handler (e.g. to offer a retry).
func (m *Model) actionWith(label string, fn func() error, onErr func(m *Model, err error) tea.Cmd) tea.Cmd {
	return m.actionFull(label, fn, onErr, nil)
}

// actionThen runs a follow-up on success (e.g. open the push dialog).
func (m *Model) actionThen(label string, fn func() error, then func(m *Model) tea.Cmd) tea.Cmd {
	return m.actionFull(label, fn, nil, then)
}

func (m *Model) actionFull(label string, fn func() error, onErr func(m *Model, err error) tea.Cmd, then func(m *Model) tea.Cmd) tea.Cmd {
	m.loading++
	m.actions++
	m.actionLabel = label
	keep := ""
	if c := m.selectedCommit(); c != nil {
		keep = c.Hash
	}
	return func() tea.Msg {
		return actionDoneMsg{label: label, err: fn(), keepHash: keep, onErr: onErr, then: then}
	}
}

func (m *Model) showToast(text string, kind int) tea.Cmd {
	m.toastID++
	id := m.toastID
	m.toast = &toast{id: id, text: text, kind: kind}
	dur := toastDuration
	if kind == 2 {
		dur = toastErrDuration
	}
	return tea.Tick(dur, func(time.Time) tea.Msg { return toastClearMsg{id: id} })
}

// copyToClipboard uses the system clipboard and falls back to OSC 52 (which
// works over SSH and on Linux without xclip/xsel) when that is unavailable.
func copyToClipboard(what, text string) tea.Cmd {
	return func() tea.Msg {
		if err := clipboard.WriteAll(text); err != nil {
			seq := "\x1b]52;c;" + base64.StdEncoding.EncodeToString([]byte(text)) + "\x07"
			if _, werr := os.Stdout.WriteString(seq); werr != nil {
				return clipboardMsg{err: err, what: what}
			}
		}
		return clipboardMsg{what: what}
	}
}

// remoteWebURL converts the origin URL into a browsable https URL.
func remoteWebURL(repo *git.Runner) string {
	out, err := repo.Run("remote", "get-url", "origin")
	if err != nil {
		return ""
	}
	u := strings.TrimSpace(out)
	u = strings.TrimSuffix(u, ".git")
	if strings.HasPrefix(u, "git@") {
		u = strings.TrimPrefix(u, "git@")
		u = strings.Replace(u, ":", "/", 1)
		u = "https://" + u
	} else if strings.HasPrefix(u, "ssh://git@") {
		u = "https://" + strings.TrimPrefix(u, "ssh://git@")
	}
	if !strings.HasPrefix(u, "http") {
		return ""
	}
	return u
}
