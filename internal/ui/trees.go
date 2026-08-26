package ui

import (
	"path"
	"sort"
	"strings"

	"github.com/jinhyo-dev/gitpad/internal/git"
)

// ---- branch tree -------------------------------------------------------

type bkind int

const (
	bHead bkind = iota
	bSection
	bFolder
	bLeaf
)

type bnode struct {
	kind   bkind
	depth  int
	label  string
	key    string
	branch *git.Branch
	count  int
}

type trie struct {
	name     string
	children map[string]*trie
	leaf     *git.Branch
	count    int
}

func newTrie(name string) *trie { return &trie{name: name, children: map[string]*trie{}} }

func (t *trie) insert(parts []string, b *git.Branch) {
	t.count++
	if len(parts) == 1 {
		child := newTrie(parts[0])
		child.leaf = b
		child.count = 1
		t.children[parts[0]+"\x00leaf"] = child
		return
	}
	c, ok := t.children[parts[0]]
	if !ok {
		c = newTrie(parts[0])
		t.children[parts[0]] = c
	}
	c.insert(parts[1:], b)
}

func (t *trie) sortedChildren() []*trie {
	out := make([]*trie, 0, len(t.children))
	for _, c := range t.children {
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool {
		di, dj := out[i].leaf == nil, out[j].leaf == nil
		if di != dj {
			return di // folders first
		}
		return strings.ToLower(out[i].name) < strings.ToLower(out[j].name)
	})
	return out
}

func (m *Model) rebuildBranchTree() {
	prevKey := ""
	if n := m.selectedBranchNode(); n != nil {
		prevKey = n.key
	}
	var nodes []bnode
	headLabel := m.info.Head
	if m.info.Detached {
		headLabel = "detached at " + m.info.Head
	}
	nodes = append(nodes, bnode{kind: bHead, label: headLabel, key: "head"})

	addSection := func(title, key string, branches []git.Branch, splitRemote bool) {
		root := newTrie(title)
		for i := range branches {
			b := &branches[i]
			root.insert(strings.Split(b.Name, "/"), b)
		}
		nodes = append(nodes, bnode{kind: bSection, label: title, key: key, count: len(branches)})
		if !m.bexp[key] {
			return
		}
		var walk func(t *trie, depth int, prefix string)
		walk = func(t *trie, depth int, prefix string) {
			for _, c := range t.sortedChildren() {
				if c.leaf != nil {
					nodes = append(nodes, bnode{kind: bLeaf, depth: depth, label: c.name, key: prefix + "/" + c.name, branch: c.leaf})
					continue
				}
				k := prefix + "/" + c.name
				nodes = append(nodes, bnode{kind: bFolder, depth: depth, label: c.name, key: k, count: c.count})
				exp, seen := m.bexp[k]
				if !seen {
					exp = splitRemote && depth == 1 // remotes (origin/...) open by default
					m.bexp[k] = exp
				}
				if exp {
					walk(c, depth+1, k)
				}
			}
		}
		walk(root, 1, key)
	}
	addSection("Local", "local", m.refs.Locals, false)
	addSection("Remote", "remote", m.refs.Remotes, true)
	addSection("Tags", "tags", m.refs.Tags, false)
	m.bnodes = nodes

	m.bcur = 0
	if prevKey != "" {
		for i, n := range nodes {
			if n.key == prevKey {
				m.bcur = i
				break
			}
		}
	}
}

// ---- file tree ---------------------------------------------------------

type fnode struct {
	depth   int
	label   string
	key     string
	isDir   bool
	isGroup bool // "Changes" / "Unversioned files" header
	file    *git.FileChange
	count   int
}

type ftrie struct {
	name  string
	dirs  map[string]*ftrie
	files []*git.FileChange
	count int
}

func newFtrie(name string) *ftrie { return &ftrie{name: name, dirs: map[string]*ftrie{}} }

func (t *ftrie) insert(parts []string, f *git.FileChange) {
	t.count++
	if len(parts) == 1 {
		t.files = append(t.files, f)
		return
	}
	c, ok := t.dirs[parts[0]]
	if !ok {
		c = newFtrie(parts[0])
		t.dirs[parts[0]] = c
	}
	c.insert(parts[1:], f)
}

func (m *Model) rebuildFileTree() {
	var nodes []fnode
	var walk func(t *ftrie, depth int, prefix string)
	walk = func(t *ftrie, depth int, prefix string) {
		names := make([]string, 0, len(t.dirs))
		for n := range t.dirs {
			names = append(names, n)
		}
		sort.Slice(names, func(i, j int) bool { return strings.ToLower(names[i]) < strings.ToLower(names[j]) })
		for _, n := range names {
			d := t.dirs[n]
			label := d.name
			// Compact single-child directory chains: a/b/c
			for len(d.files) == 0 && len(d.dirs) == 1 {
				for _, only := range d.dirs {
					label = label + "/" + only.name
					d = only
				}
			}
			k := prefix + "/" + label
			nodes = append(nodes, fnode{depth: depth, label: label, key: k, isDir: true, count: d.count})
			if !m.fcollapsed[k] {
				walk(d, depth+1, k)
			}
		}
		files := append([]*git.FileChange(nil), t.files...)
		sort.Slice(files, func(i, j int) bool {
			return strings.ToLower(path.Base(files[i].Path)) < strings.ToLower(path.Base(files[j].Path))
		})
		for _, f := range files {
			nodes = append(nodes, fnode{depth: depth, label: path.Base(f.Path), key: "f:" + f.Path, file: f})
		}
	}
	if m.filesFor == "local" {
		// Group tracked changes separately from unversioned files.
		groups := []struct {
			title, key string
			untracked  bool
		}{{"Changes", "g:changes", false}, {"Unversioned files", "g:unversioned", true}}
		for _, g := range groups {
			root := newFtrie("")
			for i := range m.files {
				f := &m.files[i]
				if (f.Status == '?') == g.untracked {
					root.insert(strings.Split(f.Path, "/"), f)
				}
			}
			if root.count == 0 {
				continue
			}
			nodes = append(nodes, fnode{depth: 0, label: g.title, key: g.key, isDir: true, isGroup: true, count: root.count})
			if !m.fcollapsed[g.key] {
				walk(root, 1, g.key)
			}
		}
	} else {
		root := newFtrie("")
		for i := range m.files {
			f := &m.files[i]
			root.insert(strings.Split(f.Path, "/"), f)
		}
		walk(root, 0, "")
	}
	m.fnodes = nodes
	m.fcur = clamp(m.fcur, 0, maxInt(0, len(nodes)-1))
	// Prefer landing on the first file rather than a folder row.
	if m.fcur == 0 {
		for i, n := range nodes {
			if !n.isDir {
				m.fcur = i
				break
			}
		}
	}
}
