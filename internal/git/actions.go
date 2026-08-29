package git

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// ResetMode is the --soft/--mixed/--hard flag of git reset.
type ResetMode string

const (
	ResetSoft  ResetMode = "--soft"
	ResetMixed ResetMode = "--mixed"
	ResetHard  ResetMode = "--hard"
	ResetKeep  ResetMode = "--keep"
)

func (r *Runner) Checkout(ref string) error { _, err := r.RunWrite("checkout", ref); return err }
func (r *Runner) CheckoutDetached(hash string) error {
	_, err := r.RunWrite("checkout", "--detach", hash)
	return err
}
func (r *Runner) CheckoutTracking(remote string) error {
	_, err := r.RunWrite("checkout", "--track", remote)
	return err
}
func (r *Runner) Merge(ref string) error       { _, err := r.RunWrite("merge", "--no-edit", ref); return err }
func (r *Runner) Rebase(onto string) error     { _, err := r.RunWrite("rebase", onto); return err }
func (r *Runner) CherryPick(hash string) error { _, err := r.RunWrite("cherry-pick", hash); return err }
func (r *Runner) Revert(hash string) error {
	_, err := r.RunWrite("revert", "--no-edit", hash)
	return err
}
func (r *Runner) Reset(mode ResetMode, hash string) error {
	_, err := r.RunWrite("reset", string(mode), hash)
	return err
}
func (r *Runner) CreateBranch(name, at string, checkout bool) error {
	if checkout {
		_, err := r.RunWrite("checkout", "-b", name, at)
		return err
	}
	_, err := r.RunWrite("branch", name, at)
	return err
}
func (r *Runner) DeleteBranch(name string, force bool) error {
	flag := "-d"
	if force {
		flag = "-D"
	}
	_, err := r.RunWrite("branch", flag, name)
	return err
}
func (r *Runner) RenameBranch(old, new string) error {
	_, err := r.RunWrite("branch", "-m", old, new)
	return err
}
func (r *Runner) DeleteRemoteBranch(remote, name string) error {
	_, err := r.RunWrite("push", remote, "--delete", name)
	return err
}
func (r *Runner) CreateTag(name, at string) error { _, err := r.RunWrite("tag", name, at); return err }

// CreateTagAnnotated makes an annotated tag (git tag -a) with a message; an
// empty message falls back to a lightweight tag.
func (r *Runner) CreateTagAnnotated(name, at, message string) error {
	if strings.TrimSpace(message) == "" {
		return r.CreateTag(name, at)
	}
	_, err := r.RunWrite("tag", "-a", name, "-m", message, at)
	return err
}

// PushTag pushes a single tag.
func (r *Runner) PushTag(remote, name string) error {
	_, err := r.RunWrite("push", remote, "refs/tags/"+name)
	return err
}

// DeleteRemoteTag removes a tag from a remote.
func (r *Runner) DeleteRemoteTag(remote, name string) error {
	_, err := r.RunWrite("push", remote, "--delete", "refs/tags/"+name)
	return err
}
func (r *Runner) DeleteTag(name string) error { _, err := r.RunWrite("tag", "-d", name); return err }
func (r *Runner) Fetch() error                { _, err := r.RunWrite("fetch", "--all", "--prune"); return err }
func (r *Runner) Pull() error                 { _, err := r.RunWrite("pull"); return err }
func (r *Runner) Push(force bool) error {
	args := []string{"push"}
	if force {
		args = append(args, "--force-with-lease")
	}
	_, err := r.RunWrite(args...)
	return err
}
func (r *Runner) PushSetUpstream(branch string) error {
	_, err := r.RunWrite("push", "-u", "origin", branch)
	return err
}
func (r *Runner) StashPush() error {
	_, err := r.RunWrite("stash", "push", "--include-untracked")
	return err
}
func (r *Runner) StashPop() error { _, err := r.RunWrite("stash", "pop"); return err }
func (r *Runner) StageAll() error { _, err := r.RunWrite("add", "-A"); return err }
func (r *Runner) UnstageAll() error {
	_, err := r.RunWrite("reset", "-q")
	return err
}
func (r *Runner) StagePath(path string) error {
	_, err := r.RunWrite("add", "-A", "--", path)
	return err
}
func (r *Runner) UnstagePath(path string) error {
	_, err := r.RunWrite("reset", "-q", "--", path)
	return err
}
func (r *Runner) DiscardPath(fc FileChange) error {
	if fc.Status == '?' {
		return os.RemoveAll(filepath.Join(r.Dir, fc.Path))
	}
	if _, err := r.RunWrite("reset", "-q", "--", fc.Path); err != nil {
		return err
	}
	_, err := r.RunWrite("checkout", "--", fc.Path)
	return err
}
func (r *Runner) Commit(message string) error {
	if message == "" {
		return errors.New("commit message is empty")
	}
	_, err := r.RunWrite("commit", "-m", message)
	return err
}

// CommitPaths commits the working-tree state of exactly the given paths
// other staged changes are left in the index untouched.
// Untracked paths are added first so that --only can see them.
func (r *Runner) CommitPaths(message string, files []FileChange) error {
	if strings.TrimSpace(message) == "" {
		return errors.New("commit message is empty")
	}
	if len(files) == 0 {
		return errors.New("no files selected")
	}
	var paths []string
	for _, f := range files {
		if f.Status == '?' {
			if _, err := r.RunWrite("add", "--", f.Path); err != nil {
				return err
			}
		}
		paths = append(paths, f.Path)
		if f.OldPath != "" {
			paths = append(paths, f.OldPath)
		}
	}
	args := append([]string{"commit", "--only", "-m", message, "--"}, paths...)
	_, err := r.RunWrite(args...)
	return err
}

// PushOptions configure PushWith.
type PushOptions struct {
	Remote      string
	Branch      string
	SetUpstream bool
	Force       bool // --force-with-lease
	Tags        bool
}

// PushWith runs git push with the given options.
func (r *Runner) PushWith(o PushOptions) error {
	args := []string{"push"}
	if o.Force {
		args = append(args, "--force-with-lease")
	}
	if o.Tags {
		args = append(args, "--tags")
	}
	if o.SetUpstream {
		args = append(args, "-u", o.Remote, o.Branch)
	}
	_, err := r.RunWrite(args...)
	return err
}

func (r *Runner) PullRebase() error { _, err := r.RunWrite("pull", "--rebase"); return err }
func (r *Runner) PullMerge() error  { _, err := r.RunWrite("pull", "--no-rebase"); return err }
func (r *Runner) AbortState(state string) error {
	switch state {
	case "merging":
		_, err := r.RunWrite("merge", "--abort")
		return err
	case "rebasing":
		_, err := r.RunWrite("rebase", "--abort")
		return err
	case "cherry-picking":
		_, err := r.RunWrite("cherry-pick", "--abort")
		return err
	case "reverting":
		_, err := r.RunWrite("revert", "--abort")
		return err
	}
	return nil
}
func (r *Runner) ContinueState(state string) error {
	switch state {
	case "rebasing":
		_, err := r.RunWrite("rebase", "--continue")
		return err
	case "cherry-picking":
		_, err := r.RunWrite("cherry-pick", "--continue")
		return err
	case "reverting":
		_, err := r.RunWrite("revert", "--continue")
		return err
	case "merging":
		_, err := r.RunWrite("commit", "--no-edit")
		return err
	}
	return nil
}

func exists(parts ...string) bool {
	_, err := os.Stat(filepath.Join(parts...))
	return err == nil
}

func asErr(err error, target **Error) bool { return errors.As(err, target) }
