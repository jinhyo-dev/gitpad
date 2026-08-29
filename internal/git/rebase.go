package git

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// RebaseAction is one verb of an interactive-rebase todo list.
type RebaseAction string

const (
	Pick   RebaseAction = "pick"
	Reword RebaseAction = "reword"
	Edit   RebaseAction = "edit"
	Squash RebaseAction = "squash"
	Fixup  RebaseAction = "fixup"
	Drop   RebaseAction = "drop"
)

// RebaseStep is one line of the plan, in execution (oldest-first) order.
type RebaseStep struct {
	Hash    string
	Subject string
	Action  RebaseAction
	Message string // new message for Reword
}

// RebasePlan describes what can be rebased from a starting commit up to HEAD.
type RebasePlan struct {
	Base    string // parent of the oldest commit; "" when rebasing from the root
	Root    bool
	Commits []Commit // newest first, as in the log
}

// PlanRebase collects the first-parent commits from `from` (inclusive) up to
// HEAD and refuses ranges that git rebase -i cannot replay linearly.
func (r *Runner) PlanRebase(from string) (RebasePlan, error) {
	if out, err := r.Run("status", "--porcelain", "--untracked-files=no"); err == nil && strings.TrimSpace(out) != "" {
		return RebasePlan{}, errors.New("working tree has uncommitted changes — commit or stash them first")
	}
	if _, err := r.Run("merge-base", "--is-ancestor", from, "HEAD"); err != nil {
		return RebasePlan{}, errors.New("the commit is not an ancestor of HEAD")
	}
	plan := RebasePlan{}
	parents, _ := r.Run("rev-list", "--parents", "-n", "1", from)
	if len(strings.Fields(parents)) < 2 {
		plan.Root = true
	} else {
		plan.Base = strings.Fields(parents)[1]
	}
	rng := from + "^..HEAD"
	if plan.Root {
		rng = "HEAD"
	}
	if merges, err := r.Run("rev-list", "--merges", rng); err == nil && strings.TrimSpace(merges) != "" {
		return RebasePlan{}, errors.New("the range contains merge commits — pick a later commit")
	}
	commits, err := r.Log(LogOptions{Extra: []string{rng}, Limit: 1000})
	if err != nil {
		return RebasePlan{}, err
	}
	if len(commits) == 0 {
		return RebasePlan{}, errors.New("nothing to rebase")
	}
	plan.Commits = commits
	return plan, nil
}

// RebaseInteractive runs `git rebase -i` with the given steps (oldest first).
// The todo list is written by gitpad itself (see WriteTodoArg), so no editor
// is ever opened; rewords become `exec git commit --amend -F <file>` lines.
func (r *Runner) RebaseInteractive(plan RebasePlan, steps []RebaseStep) error {
	if len(steps) == 0 {
		return errors.New("empty rebase plan")
	}
	dir, err := os.MkdirTemp("", "gitpad-rebase-")
	if err != nil {
		return err
	}
	// The directory must outlive the git process; git reads exec files lazily.
	defer os.RemoveAll(dir)

	var todo strings.Builder
	for i, s := range steps {
		action := s.Action
		if action == "" {
			action = Pick
		}
		if action == Reword {
			action = Pick
		}
		fmt.Fprintf(&todo, "%s %s %s\n", action, s.Hash, firstLine(s.Subject))
		if s.Action == Reword {
			msgFile := filepath.Join(dir, "msg-"+strconv.Itoa(i))
			if err := os.WriteFile(msgFile, []byte(strings.TrimSpace(s.Message)+"\n"), 0o600); err != nil {
				return err
			}
			fmt.Fprintf(&todo, "exec git commit --amend --no-verify -F %s\n", shellQuote(msgFile))
		}
	}
	todoFile := filepath.Join(dir, "todo")
	if err := os.WriteFile(todoFile, []byte(todo.String()), 0o600); err != nil {
		return err
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	args := []string{"rebase", "-i", "--no-autosquash"}
	if plan.Root {
		args = append(args, "--root")
	} else {
		args = append(args, plan.Base)
	}
	env := []string{"GIT_SEQUENCE_EDITOR=" + shellQuote(exe) + " " + WriteTodoArg + " " + shellQuote(todoFile)}
	_, err = r.RunWriteEnv(env, args...)
	return err
}

// WriteTodoArg is the hidden flag main() understands: `gitpad --write-todo
// SRC DST` copies SRC over DST (git appends DST, the todo path).
const WriteTodoArg = "--write-todo"

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	return s
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
