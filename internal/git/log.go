package git

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// RefKind classifies a decoration / ref.
type RefKind int

const (
	RefHead RefKind = iota
	RefLocal
	RefRemote
	RefTag
)

// Ref is a decoration attached to a commit.
type Ref struct {
	Name string // short name (main, origin/main, v1.0)
	Kind RefKind
}

// Commit is one row of the log.
type Commit struct {
	Hash    string
	Short   string
	Parents []string
	Author  string
	Email   string
	When    time.Time
	Refs    []Ref
	Subject string
}

// IsMerge reports whether the commit has more than one parent.
func (c Commit) IsMerge() bool { return len(c.Parents) > 1 }

// LogOptions control which commits are listed.
type LogOptions struct {
	All    bool   // every branch / tag (default: HEAD only)
	Ref    string // restrict to a ref (overrides All)
	Grep   string // matches the message OR the author name/email
	Author string // author only (used internally by Log for the OR)
	Paths  []string
	Limit  int
	Extra  []string // raw revision arguments (e.g. "@{u}..HEAD", "--not", "--remotes")
}

const logFormat = "%H%x1f%P%x1f%an%x1f%ae%x1f%at%x1f%D%x1f%s"

// Log lists commits in date order (parents always after children). A Grep
// query matches either the message or the author, which git cannot express
// in one command, so two queries are merged by date.
func (r *Runner) Log(o LogOptions) ([]Commit, error) {
	if o.Limit <= 0 {
		o.Limit = 1000
	}
	if o.Grep == "" {
		return r.logOnce(o)
	}
	byMsg, err := r.logOnce(o)
	if err != nil {
		return nil, err
	}
	a := o
	a.Grep, a.Author = "", o.Grep
	byAuthor, err := r.logOnce(a)
	if err != nil {
		return nil, err
	}
	return mergeByDate(byMsg, byAuthor), nil
}

// mergeByDate merges two date-ordered lists, dropping duplicates.
func mergeByDate(a, b []Commit) []Commit {
	out := make([]Commit, 0, len(a)+len(b))
	seen := make(map[string]bool, len(a)+len(b))
	i, j := 0, 0
	for i < len(a) || j < len(b) {
		var next Commit
		switch {
		case j >= len(b) || (i < len(a) && !a[i].When.Before(b[j].When)):
			next = a[i]
			i++
		default:
			next = b[j]
			j++
		}
		if !seen[next.Hash] {
			seen[next.Hash] = true
			out = append(out, next)
		}
	}
	return out
}

func (r *Runner) logOnce(o LogOptions) ([]Commit, error) {
	args := []string{"log", "--date-order", "--decorate=full", "--format=" + logFormat, "--max-count=" + strconv.Itoa(o.Limit)}
	switch {
	case len(o.Extra) > 0:
		args = append(args, o.Extra...)
	case o.Ref != "":
		args = append(args, o.Ref)
	case o.All:
		args = append(args, "--branches", "--remotes", "--tags", "HEAD")
	default:
		args = append(args, "HEAD")
	}
	if o.Grep != "" {
		args = append(args, "--regexp-ignore-case", "--grep="+o.Grep)
	}
	if o.Author != "" {
		args = append(args, "--regexp-ignore-case", "--author="+o.Author)
	}
	if len(o.Paths) > 0 {
		args = append(args, "--")
		args = append(args, o.Paths...)
	}
	out, err := r.Run(args...)
	if err != nil {
		var ge *Error
		if asErr(err, &ge) && (strings.Contains(ge.Stderr, "does not have any commits") || strings.Contains(ge.Stderr, "unknown revision")) {
			return nil, nil
		}
		return nil, err
	}
	var commits []Commit
	for _, line := range strings.Split(out, "\n") {
		if line == "" {
			continue
		}
		f := strings.SplitN(line, "\x1f", 7)
		if len(f) < 7 {
			continue
		}
		ts, _ := strconv.ParseInt(f[4], 10, 64)
		c := Commit{
			Hash:    f[0],
			Short:   shortHash(f[0]),
			Author:  f[2],
			Email:   f[3],
			When:    time.Unix(ts, 0),
			Refs:    parseDecorations(f[5]),
			Subject: f[6],
		}
		if f[1] != "" {
			c.Parents = strings.Split(f[1], " ")
		}
		commits = append(commits, c)
	}
	return commits, nil
}

func shortHash(h string) string {
	if len(h) > 8 {
		return h[:8]
	}
	return h
}

// parseDecorations parses a full-name %D string such as
// "HEAD -> refs/heads/main, refs/remotes/origin/main, tag: refs/tags/v1".
func parseDecorations(s string) []Ref {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	var refs []Ref
	for _, part := range strings.Split(s, ", ") {
		part = strings.TrimSpace(part)
		switch {
		case strings.HasPrefix(part, "HEAD -> "):
			refs = append(refs, Ref{Name: "HEAD", Kind: RefHead})
			part = strings.TrimPrefix(part, "HEAD -> ")
		case part == "HEAD":
			refs = append(refs, Ref{Name: "HEAD", Kind: RefHead})
			continue
		}
		part = strings.TrimPrefix(part, "tag: ")
		switch {
		case strings.HasPrefix(part, "refs/heads/"):
			refs = append(refs, Ref{Name: strings.TrimPrefix(part, "refs/heads/"), Kind: RefLocal})
		case strings.HasPrefix(part, "refs/remotes/"):
			name := strings.TrimPrefix(part, "refs/remotes/")
			if strings.HasSuffix(name, "/HEAD") {
				continue
			}
			refs = append(refs, Ref{Name: name, Kind: RefRemote})
		case strings.HasPrefix(part, "refs/tags/"):
			refs = append(refs, Ref{Name: strings.TrimPrefix(part, "refs/tags/"), Kind: RefTag})
		}
	}
	return refs
}

// Branch is a local or remote branch, or a tag.
type Branch struct {
	Name     string // short: main, origin/main, v1.0
	Full     string // refs/heads/main
	Hash     string
	Kind     RefKind
	IsHead   bool
	Remote   string // for remotes: "origin"
	Upstream string // for locals: origin/main
	Ahead    int
	Behind   int
	Gone     bool
}

// Refs is the set of all branches and tags.
type Refs struct {
	Locals  []Branch
	Remotes []Branch
	Tags    []Branch
}

// Refs lists branches and tags.
func (r *Runner) Refs() (Refs, error) {
	out, err := r.Run("for-each-ref",
		"--format=%(refname)%1f%(objectname)%1f%(HEAD)%1f%(upstream:short)%1f%(upstream:track)",
		"--sort=refname", "refs/heads", "refs/remotes", "refs/tags")
	if err != nil {
		return Refs{}, err
	}
	var refs Refs
	for _, line := range strings.Split(out, "\n") {
		if line == "" {
			continue
		}
		f := strings.SplitN(line, "\x1f", 5)
		if len(f) < 5 {
			continue
		}
		b := Branch{Full: f[0], Hash: f[1], IsHead: f[2] == "*", Upstream: f[3]}
		b.Ahead, b.Behind, b.Gone = parseTrack(f[4])
		switch {
		case strings.HasPrefix(b.Full, "refs/heads/"):
			b.Name = strings.TrimPrefix(b.Full, "refs/heads/")
			b.Kind = RefLocal
			refs.Locals = append(refs.Locals, b)
		case strings.HasPrefix(b.Full, "refs/remotes/"):
			b.Name = strings.TrimPrefix(b.Full, "refs/remotes/")
			if strings.HasSuffix(b.Name, "/HEAD") {
				continue
			}
			b.Kind = RefRemote
			if i := strings.Index(b.Name, "/"); i > 0 {
				b.Remote = b.Name[:i]
			}
			refs.Remotes = append(refs.Remotes, b)
		case strings.HasPrefix(b.Full, "refs/tags/"):
			b.Name = strings.TrimPrefix(b.Full, "refs/tags/")
			b.Kind = RefTag
			refs.Tags = append(refs.Tags, b)
		}
	}
	return refs, nil
}

func parseTrack(s string) (ahead, behind int, gone bool) {
	s = strings.Trim(s, "[]")
	if s == "gone" {
		return 0, 0, true
	}
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(p)
		if n, ok := strings.CutPrefix(p, "ahead "); ok {
			ahead, _ = strconv.Atoi(n)
		} else if n, ok := strings.CutPrefix(p, "behind "); ok {
			behind, _ = strconv.Atoi(n)
		}
	}
	return
}

// RepoInfo summarizes HEAD.
type RepoInfo struct {
	Name     string
	Head     string // branch name, or short hash when detached
	HeadHash string
	Detached bool
	Upstream string
	Ahead    int
	Behind   int
	State    string // "merging", "rebasing", "cherry-picking", "reverting" or ""
}

// Info reads the current HEAD state.
func (r *Runner) Info() RepoInfo {
	info := RepoInfo{Name: r.Name()}
	if out, err := r.Run("rev-parse", "HEAD"); err == nil {
		info.HeadHash = strings.TrimSpace(out)
	}
	if out, err := r.Run("symbolic-ref", "--short", "-q", "HEAD"); err == nil && strings.TrimSpace(out) != "" {
		info.Head = strings.TrimSpace(out)
	} else {
		info.Detached = true
		info.Head = shortHash(info.HeadHash)
		if info.Head == "" {
			info.Head = "(no commits)"
		}
	}
	if !info.Detached {
		if out, err := r.Run("rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{u}"); err == nil {
			info.Upstream = strings.TrimSpace(out)
			if out, err := r.Run("rev-list", "--left-right", "--count", "HEAD...@{u}"); err == nil {
				f := strings.Fields(out)
				if len(f) == 2 {
					info.Ahead, _ = strconv.Atoi(f[0])
					info.Behind, _ = strconv.Atoi(f[1])
				}
			}
		}
	}
	gitDir := ""
	if out, err := r.Run("rev-parse", "--git-dir"); err == nil {
		gitDir = strings.TrimSpace(out)
	}
	if gitDir != "" {
		switch {
		case exists(r.Dir, gitDir, "MERGE_HEAD"):
			info.State = "merging"
		case exists(r.Dir, gitDir, "rebase-merge") || exists(r.Dir, gitDir, "rebase-apply"):
			info.State = "rebasing"
		case exists(r.Dir, gitDir, "CHERRY_PICK_HEAD"):
			info.State = "cherry-picking"
		case exists(r.Dir, gitDir, "REVERT_HEAD"):
			info.State = "reverting"
		}
	}
	return info
}

// FileChange is one changed path.
type FileChange struct {
	Status   byte // A M D R C T U ? (untracked) !
	Path     string
	OldPath  string // for renames
	Staged   bool
	Unstaged bool
	Conflict bool
}

// Status lists working tree changes (staged, unstaged and untracked).
func (r *Runner) Status() ([]FileChange, error) {
	out, err := r.Run("status", "--porcelain=v1", "-z", "--untracked-files=all")
	if err != nil {
		return nil, err
	}
	parts := strings.Split(out, "\x00")
	var files []FileChange
	for i := 0; i < len(parts); i++ {
		p := parts[i]
		if len(p) < 4 {
			continue
		}
		x, y := p[0], p[1]
		fc := FileChange{Path: p[3:]}
		switch {
		case x == '?' && y == '?':
			fc.Status = '?'
			fc.Unstaged = true
		case x == 'U' || y == 'U' || (x == 'A' && y == 'A') || (x == 'D' && y == 'D'):
			fc.Status = 'U'
			fc.Conflict = true
			fc.Unstaged = true
		default:
			if x != ' ' && x != '?' {
				fc.Staged = true
				fc.Status = x
			}
			if y != ' ' {
				fc.Unstaged = true
				if fc.Status == 0 {
					fc.Status = y
				}
			}
		}
		if x == 'R' || x == 'C' {
			if i+1 < len(parts) {
				fc.OldPath = parts[i+1]
				i++
			}
		}
		files = append(files, fc)
	}
	return files, nil
}

// CommitFiles lists the files changed by a commit (vs its first parent).
func (r *Runner) CommitFiles(c Commit) ([]FileChange, error) {
	var out string
	var err error
	if len(c.Parents) > 0 {
		out, err = r.Run("diff", "--name-status", "-M", "-z", c.Parents[0], c.Hash)
	} else {
		out, err = r.Run("diff-tree", "--root", "-r", "-M", "--no-commit-id", "--name-status", "-z", c.Hash)
	}
	if err != nil {
		return nil, err
	}
	return parseNameStatus(out), nil
}

func parseNameStatus(out string) []FileChange {
	parts := strings.Split(out, "\x00")
	var files []FileChange
	for i := 0; i+1 < len(parts); i += 2 {
		st := parts[i]
		if st == "" {
			break
		}
		fc := FileChange{Status: st[0], Path: parts[i+1]}
		if (st[0] == 'R' || st[0] == 'C') && i+2 < len(parts) {
			fc.OldPath = parts[i+1]
			fc.Path = parts[i+2]
			i++
		}
		files = append(files, fc)
	}
	return files
}

// CommitDetails is the full information for the details pane.
type CommitDetails struct {
	Hash      string
	Author    string
	Email     string
	AuthorAt  time.Time
	Committer string
	CommitAt  time.Time
	Parents   []string
	Refs      []Ref
	Message   string // full message (subject + body)
}

// Details fetches full commit info.
func (r *Runner) Details(hash string) (*CommitDetails, error) {
	out, err := r.Run("show", "-s", "--decorate=full", "--format=%H%x1f%an%x1f%ae%x1f%at%x1f%cn%x1f%ct%x1f%P%x1f%D%x1f%B", hash)
	if err != nil {
		return nil, err
	}
	f := strings.SplitN(out, "\x1f", 9)
	if len(f) < 9 {
		return nil, fmt.Errorf("unexpected git show output")
	}
	at, _ := strconv.ParseInt(f[3], 10, 64)
	ct, _ := strconv.ParseInt(f[5], 10, 64)
	d := &CommitDetails{
		Hash: f[0], Author: f[1], Email: f[2], AuthorAt: time.Unix(at, 0),
		Committer: f[4], CommitAt: time.Unix(ct, 0),
		Refs:    parseDecorations(f[7]),
		Message: strings.TrimRight(f[8], "\n"),
	}
	if f[6] != "" {
		d.Parents = strings.Split(f[6], " ")
	}
	return d, nil
}

// CommitDiff returns the unified diff of one path in a commit.
func (r *Runner) CommitDiff(c Commit, path string) (string, error) {
	if len(c.Parents) > 0 {
		return r.Run("diff", "--no-color", "-M", c.Parents[0], c.Hash, "--", path)
	}
	return r.Run("show", "--no-color", "--format=", "-M", "--root", c.Hash, "--", path)
}

// WorktreeDiff returns the diff of a working tree change against HEAD.
func (r *Runner) WorktreeDiff(fc FileChange) (string, error) {
	if fc.Status == '?' {
		out, err := r.Run("diff", "--no-color", "--no-index", "--", "/dev/null", fc.Path)
		// --no-index exits 1 when files differ, which is expected.
		if err != nil && out != "" {
			return out, nil
		}
		return out, err
	}
	if out, err := r.Run("rev-parse", "--verify", "-q", "HEAD"); err != nil || strings.TrimSpace(out) == "" {
		return r.Run("diff", "--no-color", "--cached", "-M", "--", fc.Path)
	}
	return r.Run("diff", "--no-color", "-M", "HEAD", "--", fc.Path)
}

// ResolveHash expands a hash prefix to a full commit hash.
func (r *Runner) ResolveHash(prefix string) (string, bool) {
	out, err := r.Run("rev-parse", "--verify", "-q", prefix+"^{commit}")
	if err != nil {
		return "", false
	}
	return strings.TrimSpace(out), true
}

// RecentMessages returns the last n commit messages written by the current
// user (falling back to anyone) for the message-history feature.
func (r *Runner) RecentMessages(n int) []string {
	args := []string{"log", "--format=%B%x1e", "--max-count=" + strconv.Itoa(n)}
	if email := r.UserEmail(); email != "" {
		args = append(args, "--author="+email)
	}
	out, err := r.Run(args...)
	if err != nil || strings.TrimSpace(out) == "" {
		out, err = r.Run("log", "--format=%B%x1e", "--max-count="+strconv.Itoa(n))
		if err != nil {
			return nil
		}
	}
	var msgs []string
	seen := map[string]bool{}
	for _, m := range strings.Split(out, "\x1e") {
		m = strings.TrimSpace(m)
		if m == "" || seen[m] {
			continue
		}
		seen[m] = true
		msgs = append(msgs, m)
	}
	return msgs
}

// UserEmail returns git's configured user.email.
func (r *Runner) UserEmail() string {
	out, _ := r.Run("config", "--get", "user.email")
	return strings.TrimSpace(out)
}

// PullRebaseDefault reports whether pull.rebase is enabled in config.
func (r *Runner) PullRebaseDefault() bool {
	out, _ := r.Run("config", "--get", "pull.rebase")
	v := strings.TrimSpace(strings.ToLower(out))
	return v == "true" || v == "merges" || v == "interactive"
}

// Remotes lists remote names.
func (r *Runner) Remotes() []string {
	out, _ := r.Run("remote")
	return strings.Fields(out)
}

// HeadMessage returns the full message of HEAD.
func (r *Runner) HeadMessage() string {
	out, _ := r.Run("log", "-1", "--format=%B")
	return strings.TrimRight(out, "\n")
}
