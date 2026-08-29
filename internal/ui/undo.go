package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/jinhyo-dev/gitpad/internal/git"
)

// Undo: every history-changing operation gitpad runs records where HEAD was
// beforehand; `u` walks that stack back. The old commits are always still in
// the reflog, so an undo is just a reset / checkout / re-create to a known
// point — never data loss beyond what git itself would allow.

type undoKind int

const (
	undoSkip           undoKind = iota // not undoable (fetch, push, stash…)
	undoSoft                           // commit-like: reset --soft keeps the changes staged
	undoKeep                           // rebase/reset/merge…: reset --keep (refuses to clobber local edits)
	undoRecreateBranch                 // deleted branch → git branch name hash
	undoRecreateTag                    // deleted tag → git tag name hash
	undoDeleteTag                      // created tag → git tag -d name
)

type undoPoint struct {
	label    string
	kind     undoKind
	head     string // HEAD hash before the operation
	branch   string // branch name before (or short hash when detached)
	detached bool
	name     string // ref name for recreate/delete kinds
	hash     string // ref target for recreate kinds
}

const undoMax = 50

// undoKindFor classifies an action by its label prefix.
func undoKindFor(label string) undoKind {
	l := strings.ToLower(label)
	switch {
	case strings.HasPrefix(l, "undo"):
		return undoSkip
	case strings.HasPrefix(l, "commit"):
		return undoSoft
	case strings.HasPrefix(l, "rebase"), strings.HasPrefix(l, "reset"), strings.HasPrefix(l, "cherry-pick"),
		strings.HasPrefix(l, "revert"), strings.HasPrefix(l, "merge"), strings.HasPrefix(l, "pull"),
		strings.HasPrefix(l, "checkout"), strings.HasPrefix(l, "create"), strings.HasPrefix(l, "continue"),
		strings.HasPrefix(l, "abort"), strings.HasPrefix(l, "rename"):
		return undoKeep
	}
	return undoSkip
}

// pushUndo records a point after a successful operation.
func (m *Model) pushUndo(p undoPoint) {
	if p.kind == undoSkip {
		return
	}
	m.undo = append(m.undo, p)
	if len(m.undo) > undoMax {
		m.undo = m.undo[len(m.undo)-undoMax:]
	}
}

// describeUndo explains what undoing the top point will do.
func (m *Model) describeUndo(p undoPoint) (body string, ok bool) {
	short := func(h string) string { return h[:minInt(8, len(h))] }
	switch p.kind {
	case undoRecreateBranch:
		return fmt.Sprintf("Re-create branch %s at %s.", p.name, short(p.hash)), true
	case undoRecreateTag:
		return fmt.Sprintf("Re-create tag %s at %s.", p.name, short(p.hash)), true
	case undoDeleteTag:
		return fmt.Sprintf("Delete tag %s again.", p.name), true
	}
	switch {
	case !p.detached && m.info.Head != p.branch:
		return fmt.Sprintf("Switch back to %s.", p.branch), true
	case p.detached && !m.info.Detached:
		return fmt.Sprintf("Return to detached HEAD at %s.", short(p.head)), true
	case m.info.HeadHash != p.head:
		how := "reset --keep (local edits are preserved; refused if they would be overwritten)"
		if p.kind == undoSoft {
			how = "reset --soft — the committed changes stay staged in the working tree"
		}
		return fmt.Sprintf("%s goes back to %s via %s.", m.currentBranch(), short(p.head), how), true
	}
	return "", false
}

// undoLast asks, then reverts the most recent recorded operation.
func (m *Model) undoLast() tea.Cmd {
	for len(m.undo) > 0 {
		p := m.undo[len(m.undo)-1]
		body, ok := m.describeUndo(p)
		if ok {
			m.confirm("Undo "+p.label+"?", body, "Undo", false, func(m *Model) tea.Cmd {
				m.undo = m.undo[:len(m.undo)-1]
				repo := m.repo
				return m.action("Undo "+p.label, func() error { return runUndo(repo, p) })
			})
			return nil
		}
		m.undo = m.undo[:len(m.undo)-1] // already at that state; skip it
	}
	return m.showToast("Nothing to undo", 0)
}

func runUndo(repo *git.Runner, p undoPoint) error {
	switch p.kind {
	case undoRecreateBranch:
		return repo.CreateBranch(p.name, p.hash, false)
	case undoRecreateTag:
		return repo.CreateTag(p.name, p.hash)
	case undoDeleteTag:
		return repo.DeleteTag(p.name)
	}
	info := repo.Info()
	switch {
	case !p.detached && info.Head != p.branch:
		if err := repo.Checkout(p.branch); err != nil {
			// The branch may be gone (e.g. undoing a rename) — fall back to the commit.
			return repo.CheckoutDetached(p.head)
		}
		return nil
	case p.detached && !info.Detached:
		return repo.CheckoutDetached(p.head)
	}
	mode := git.ResetKeep
	if p.kind == undoSoft {
		mode = git.ResetSoft
	}
	return repo.Reset(mode, p.head)
}
