package git

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Selection describes what to commit for one path: the whole working-tree
// change, or only some of its hunks (indices into the hunks of
// `git diff HEAD -- path`, in order).
type Selection struct {
	Path      string
	OldPath   string
	Untracked bool
	Hunks     []int // nil = whole file
}

// CommitSelection commits exactly the selected files and hunks and nothing
// else, even when the index holds other staged changes. It runs the real
// `git commit`, so hooks and signing behave normally.
//
// The index is rebuilt from HEAD plus the selection before committing; if
// anything fails the previous index file is restored byte for byte. Other
// staged-but-unselected changes end up unstaged (still in the working tree).
func (r *Runner) CommitSelection(message string, sel []Selection) (err error) {
	if strings.TrimSpace(message) == "" {
		return errors.New("commit message is empty")
	}
	if len(sel) == 0 {
		return errors.New("nothing selected")
	}
	gitDir, gerr := r.Run("rev-parse", "--git-dir")
	if gerr != nil {
		return gerr
	}
	indexPath := filepath.Join(r.Dir, strings.TrimSpace(gitDir), "index")
	if filepath.IsAbs(strings.TrimSpace(gitDir)) {
		indexPath = filepath.Join(strings.TrimSpace(gitDir), "index")
	}
	backup, _ := os.ReadFile(indexPath) // may not exist in a brand-new repo
	restore := func() {
		if backup != nil {
			_ = os.WriteFile(indexPath, backup, 0o644)
		} else {
			_ = os.Remove(indexPath)
		}
	}
	defer func() {
		if err != nil {
			restore()
		}
	}()

	// 1. Index := HEAD (working tree untouched).
	if _, e := r.Run("rev-parse", "--verify", "-q", "HEAD"); e == nil {
		if _, e := r.RunWrite("reset", "-q"); e != nil {
			return e
		}
	} else if _, e := r.RunWrite("read-tree", "--empty"); e != nil {
		return e
	}

	// 2. Add the selection.
	for _, s := range sel {
		if s.Hunks == nil || s.Untracked {
			args := []string{"add", "-A", "--", s.Path}
			if s.OldPath != "" {
				args = append(args, s.OldPath)
			}
			if _, e := r.RunWrite(args...); e != nil {
				return e
			}
			continue
		}
		patch, e := r.SelectedPatch(s.Path, s.Hunks)
		if e != nil {
			return e
		}
		if patch == "" {
			continue
		}
		if e := r.applyCached(patch); e != nil {
			return fmt.Errorf("%s: %w", s.Path, e)
		}
	}

	// 3. Commit the index through the normal path (hooks, signing).
	if _, e := r.RunWrite("commit", "-m", message); e != nil {
		return e
	}
	return nil
}

// SelectedPatch returns `git diff HEAD -- path` reduced to the given hunks.
func (r *Runner) SelectedPatch(path string, hunks []int) (string, error) {
	diff, err := r.Run("diff", "--no-color", "--no-ext-diff", "HEAD", "--", path)
	if err != nil {
		return "", err
	}
	return FilterHunks(diff, hunks), nil
}

// FilterHunks keeps only the hunks whose 0-based index is in keep. The file
// header is preserved; an empty result means no hunk matched.
func FilterHunks(diff string, keep []int) string {
	want := map[int]bool{}
	for _, k := range keep {
		want[k] = true
	}
	lines := strings.Split(strings.TrimRight(diff, "\n"), "\n")
	var header, out []string
	inHeader := true
	hunk := -1
	include := false
	for _, l := range lines {
		if strings.HasPrefix(l, "@@") {
			inHeader = false
			hunk++
			include = want[hunk]
		}
		switch {
		case inHeader:
			header = append(header, l)
		case include:
			out = append(out, l)
		}
	}
	if len(out) == 0 {
		return ""
	}
	return strings.Join(append(header, out...), "\n") + "\n"
}

// HunkCount counts the hunks in a unified diff.
func HunkCount(diff string) int {
	n := 0
	for _, l := range strings.Split(diff, "\n") {
		if strings.HasPrefix(l, "@@") {
			n++
		}
	}
	return n
}

func (r *Runner) applyCached(patch string) error {
	f, err := os.CreateTemp("", "gitpad-hunk-*.patch")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err := f.WriteString(patch); err != nil {
		f.Close()
		return err
	}
	f.Close()
	_, err = r.RunWrite("apply", "--cached", "--recount", f.Name())
	return err
}
