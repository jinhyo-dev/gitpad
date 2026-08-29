package git

import (
	"os"
	"path/filepath"
	"strings"
)

// Conflict is one <<<<<<< … >>>>>>> block inside a file with merge markers.
type Conflict struct {
	Start       int      // line index of "<<<<<<<"
	End         int      // line index of ">>>>>>>"
	Ours        []string // lines between <<<<<<< and ======= (or |||||||)
	Base        []string // lines between ||||||| and ======= (diff3 style), nil otherwise
	Theirs      []string // lines between ======= and >>>>>>>
	OursLabel   string
	TheirsLabel string
}

// Choice is how a conflict gets resolved.
type Choice int

const (
	Unresolved Choice = iota
	KeepOurs
	KeepTheirs
	KeepBoth        // ours, then theirs
	KeepBothReverse // theirs, then ours
)

// ParseConflicts splits file content into lines and finds the marker blocks.
func ParseConflicts(content string) (lines []string, conflicts []Conflict) {
	lines = strings.Split(content, "\n")
	var cur *Conflict
	section := 0 // 1 ours, 2 base, 3 theirs
	for i, l := range lines {
		switch {
		case strings.HasPrefix(l, "<<<<<<<") && cur == nil:
			cur = &Conflict{Start: i, OursLabel: strings.TrimSpace(strings.TrimPrefix(l, "<<<<<<<"))}
			section = 1
		case cur != nil && strings.HasPrefix(l, "|||||||") && section == 1:
			cur.Base = []string{}
			section = 2
		case cur != nil && strings.HasPrefix(l, "=======") && (section == 1 || section == 2):
			section = 3
		case cur != nil && strings.HasPrefix(l, ">>>>>>>") && section == 3:
			cur.End = i
			cur.TheirsLabel = strings.TrimSpace(strings.TrimPrefix(l, ">>>>>>>"))
			conflicts = append(conflicts, *cur)
			cur = nil
			section = 0
		case cur != nil:
			switch section {
			case 1:
				cur.Ours = append(cur.Ours, l)
			case 2:
				cur.Base = append(cur.Base, l)
			case 3:
				cur.Theirs = append(cur.Theirs, l)
			}
		}
	}
	return lines, conflicts
}

// Resolve rebuilds the file applying choices (index-aligned with conflicts).
// Unresolved conflicts keep their markers; allResolved reports whether none
// remain.
func Resolve(lines []string, conflicts []Conflict, choices []Choice) (content string, allResolved bool) {
	var out []string
	allResolved = true
	next := 0
	for i := 0; i < len(lines); i++ {
		if next < len(conflicts) && i == conflicts[next].Start {
			c := conflicts[next]
			ch := Unresolved
			if next < len(choices) {
				ch = choices[next]
			}
			switch ch {
			case KeepOurs:
				out = append(out, c.Ours...)
			case KeepTheirs:
				out = append(out, c.Theirs...)
			case KeepBoth:
				out = append(out, c.Ours...)
				out = append(out, c.Theirs...)
			case KeepBothReverse:
				out = append(out, c.Theirs...)
				out = append(out, c.Ours...)
			default:
				allResolved = false
				out = append(out, lines[c.Start:c.End+1]...)
			}
			i = c.End
			next++
			continue
		}
		out = append(out, lines[i])
	}
	return strings.Join(out, "\n"), allResolved
}

// ReadWorktreeFile reads a file from the working tree.
func (r *Runner) ReadWorktreeFile(path string) (string, error) {
	b, err := os.ReadFile(filepath.Join(r.Dir, path))
	return string(b), err
}

// WriteWorktreeFile writes a file in the working tree.
func (r *Runner) WriteWorktreeFile(path, content string) error {
	return os.WriteFile(filepath.Join(r.Dir, path), []byte(content), 0o644)
}

// ConflictedFiles lists paths that currently have unmerged entries.
func (r *Runner) ConflictedFiles() ([]string, error) {
	out, err := r.Run("diff", "--name-only", "--diff-filter=U", "-z")
	if err != nil {
		return nil, err
	}
	var paths []string
	for _, p := range strings.Split(out, "\x00") {
		if p != "" {
			paths = append(paths, p)
		}
	}
	return paths, nil
}
