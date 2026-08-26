package ui

import (
	"crypto/sha1"
	"fmt"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/jinhyo/gitpad/internal/git"
)

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

func copyToClipboard(what, text string) tea.Cmd {
	return func() tea.Msg {
		return clipboardMsg{err: clipboard.WriteAll(text), what: what}
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
