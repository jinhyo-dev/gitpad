package ui

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/jinhyo-dev/gitpad/internal/ci"
	"github.com/jinhyo-dev/gitpad/internal/git"
)

// ---- filters & loading -------------------------------------------------

func (m *Model) reload() tea.Cmd {
	keep := ""
	if c := m.selectedCommit(); c != nil {
		keep = c.Hash
	}
	return m.loadAll(keep)
}

func (m *Model) applyFilter() tea.Cmd {
	m.lcur, m.lscroll = 0, 0
	m.diff = nil
	m.logOpts.Limit = 1000
	return m.loadAll("")
}

func (m *Model) loadMore() tea.Cmd {
	if m.loading > 0 {
		return nil
	}
	m.logOpts.Limit += 1000
	return m.reload()
}

func (m *Model) showRefInLog(ref string) tea.Cmd {
	m.logOpts.Ref = ref
	m.logOpts.All = ref == ""
	m.focus = PanelLog
	return m.applyFilter()
}

func (m *Model) showPathHistory(path string) tea.Cmd {
	m.logOpts.Paths = []string{path}
	m.focus = PanelLog
	return m.applyFilter()
}

// ---- shared operations -------------------------------------------------

func (m *Model) checkoutBranch(b *git.Branch) tea.Cmd {
	switch b.Kind {
	case git.RefLocal:
		return m.action("Checkout "+b.Name, func() error { return m.repo.Checkout(b.Name) })
	case git.RefRemote:
		local := strings.TrimPrefix(b.Name, b.Remote+"/")
		for _, l := range m.refs.Locals {
			if l.Name == local {
				return m.action("Checkout "+local, func() error { return m.repo.Checkout(local) })
			}
		}
		return m.action("Checkout "+local, func() error { return m.repo.CheckoutTracking(b.Name) })
	default:
		return m.action("Checkout "+b.Name, func() error { return m.repo.CheckoutDetached(b.Name) })
	}
}

func (m *Model) deleteRef(b *git.Branch) tea.Cmd {
	switch b.Kind {
	case git.RefLocal:
		if b.IsHead {
			return m.showToast("cannot delete the current branch", 2)
		}
		name := b.Name
		m.confirm("Delete branch "+name+"?", "The branch is removed locally. Commits stay reachable from the reflog for a while.", "Delete", true, func(m *Model) tea.Cmd {
			return m.actionWith("Delete "+name, func() error { return m.repo.DeleteBranch(name, false) },
				func(m *Model, err error) tea.Cmd {
					if strings.Contains(err.Error(), "not fully merged") {
						m.confirm("Branch "+name+" is not fully merged", "Force delete anyway? Unmerged commits will only be reachable from the reflog.", "Force delete", true, func(m *Model) tea.Cmd {
							return m.action("Force delete "+name, func() error { return m.repo.DeleteBranch(name, true) })
						})
						return nil
					}
					return m.showToast(err.Error(), 2)
				})
		})
	case git.RefRemote:
		remote, name := b.Remote, strings.TrimPrefix(b.Name, b.Remote+"/")
		m.confirm("Delete "+b.Name+" on "+remote+"?", "This runs git push "+remote+" --delete "+name+" and affects everyone using the remote.", "Delete remote branch", true, func(m *Model) tea.Cmd {
			return m.action("Delete "+b.Name, func() error { return m.repo.DeleteRemoteBranch(remote, name) })
		})
	default:
		name := b.Name
		m.confirm("Delete tag "+name+"?", "Only the local tag is removed.", "Delete", true, func(m *Model) tea.Cmd {
			return m.action("Delete tag "+name, func() error { return m.repo.DeleteTag(name) })
		})
	}
	return nil
}

func (m *Model) pushCmd(force bool) tea.Cmd {
	if m.info.Detached {
		return m.showToast("cannot push a detached HEAD", 2)
	}
	if m.info.Upstream == "" {
		branch := m.info.Head
		m.confirm("Push "+branch+"?", "The branch has no upstream yet. gitpad will run git push -u origin "+branch+".", "Push", false, func(m *Model) tea.Cmd {
			return m.action("Push "+branch, func() error { return m.repo.PushSetUpstream(branch) })
		})
		return nil
	}
	if force {
		m.confirm("Force push "+m.info.Head+"?", "Runs git push --force-with-lease. Remote commits not in your history will be overwritten.", "Force push", true, func(m *Model) tea.Cmd {
			return m.action("Force push", func() error { return m.repo.Push(true) })
		})
		return nil
	}
	return m.action("Push", func() error { return m.repo.Push(false) })
}

func (m *Model) toggleStage(fc git.FileChange) tea.Cmd {
	if fc.Staged && !fc.Unstaged {
		return m.action("Unstage "+fc.Path, func() error { return m.repo.UnstagePath(fc.Path) })
	}
	return m.action("Stage "+fc.Path, func() error { return m.repo.StagePath(fc.Path) })
}

func (m *Model) discardPrompt(fc git.FileChange) tea.Cmd {
	m.confirm("Discard changes in "+fc.Path+"?", "Local modifications are lost. This cannot be undone.", "Discard", true, func(m *Model) tea.Cmd {
		return m.action("Discard "+fc.Path, func() error { return m.repo.DiscardPath(fc) })
	})
	return nil
}

func (m *Model) newBranchPrompt(at, atLabel string, checkout bool) tea.Cmd {
	return m.prompt("New branch", "Create a branch at "+atLabel+".", "branch name", "", func(m *Model, name string) tea.Cmd {
		return m.action("Create "+name, func() error { return m.repo.CreateBranch(name, at, checkout) })
	})
}

func (m *Model) newTagPrompt(at, atLabel string) tea.Cmd {
	return m.prompt("New tag", "Tag "+atLabel+".", "tag name (e.g. v1.2.0)", "", func(m *Model, name string) tea.Cmd {
		return m.tagMessagePrompt(name, at, false)
	})
}

// tagMessagePrompt asks for an optional annotation, creates the tag and
// then offers to push it.
func (m *Model) tagMessagePrompt(name, at string, offerPush bool) tea.Cmd {
	return m.promptOptional("Tag "+name, "Message for an annotated tag — leave empty for a lightweight tag.", "release notes / message", func(m *Model, msg string) tea.Cmd {
		return m.actionThen("Tag "+name, func() error { return m.repo.CreateTagAnnotated(name, at, msg) }, func(m *Model) tea.Cmd {
			if !offerPush {
				return nil
			}
			remote := m.defaultRemote()
			if remote == "" {
				return nil
			}
			m.confirm("Push "+name+" to "+remote+"?", "Pushing the tag triggers release workflows that watch for tags.", "Push tag", false, func(m *Model) tea.Cmd {
				return m.action("Push "+name, func() error { return m.repo.PushTag(remote, name) })
			})
			return nil
		})
	})
}

func (m *Model) defaultRemote() string {
	remotes := m.repo.Remotes()
	for _, r := range remotes {
		if r == "origin" {
			return r
		}
	}
	if len(remotes) > 0 {
		return remotes[0]
	}
	return ""
}

// semver is a parsed vMAJOR.MINOR.PATCH tag.
type semver struct {
	major, minor, patch int
	prefix              string // "v" or ""
}

func parseSemver(s string) (semver, bool) {
	v := semver{}
	if strings.HasPrefix(s, "v") {
		v.prefix = "v"
		s = s[1:]
	}
	parts := strings.Split(s, ".")
	if len(parts) != 3 {
		return v, false
	}
	nums := [3]int{}
	for i, p := range parts {
		n := 0
		if p == "" {
			return v, false
		}
		for _, r := range p {
			if r < '0' || r > '9' {
				return v, false
			}
			n = n*10 + int(r-'0')
		}
		nums[i] = n
	}
	v.major, v.minor, v.patch = nums[0], nums[1], nums[2]
	return v, true
}

func (v semver) String() string {
	return fmt.Sprintf("%s%d.%d.%d", v.prefix, v.major, v.minor, v.patch)
}

func (v semver) less(o semver) bool {
	if v.major != o.major {
		return v.major < o.major
	}
	if v.minor != o.minor {
		return v.minor < o.minor
	}
	return v.patch < o.patch
}

// latestSemver finds the highest semver tag, or ok=false when there is none.
func latestSemver(tags []git.Branch) (semver, bool) {
	var best semver
	found := false
	for _, t := range tags {
		v, ok := parseSemver(t.Name)
		if !ok {
			continue
		}
		if !found || best.less(v) {
			best, found = v, true
		}
	}
	return best, found
}

// versionTagMenu proposes the next patch / minor / major version tag at HEAD.
func (m *Model) versionTagMenu() tea.Cmd {
	latest, ok := latestSemver(m.refs.Tags)
	title := "New version tag"
	var choices []struct {
		v    semver
		desc string
	}
	if ok {
		title = "New version — latest " + latest.String()
		choices = []struct {
			v    semver
			desc string
		}{
			{semver{latest.major, latest.minor, latest.patch + 1, latest.prefix}, "patch · bug fixes"},
			{semver{latest.major, latest.minor + 1, 0, latest.prefix}, "minor · new features"},
			{semver{latest.major + 1, 0, 0, latest.prefix}, "major · breaking / stable"},
		}
	} else {
		title = "New version — no version tags yet"
		choices = []struct {
			v    semver
			desc string
		}{
			{semver{0, 1, 0, "v"}, "first pre-release"},
			{semver{1, 0, 0, "v"}, "first stable release"},
		}
	}
	var items []menuItem
	for _, c := range choices {
		name := c.v.String()
		items = append(items, menuItem{label: pad(name, 10) + c.desc, run: func(m *Model) tea.Cmd { return m.tagMessagePrompt(name, "HEAD", true) }})
	}
	items = append(items, sep(), menuItem{label: "Custom…", key: "c", run: func(m *Model) tea.Cmd {
		return m.prompt("New tag", "Tag HEAD with a custom name.", "tag name", "", func(m *Model, name string) tea.Cmd {
			return m.tagMessagePrompt(name, "HEAD", true)
		})
	}})
	mn := &menu{title: title, items: items}
	mn.layout()
	m.openMenu(mn, (m.width-mn.w)/2, (m.height-mn.h)/2)
	if ok {
		mn.cur = 1 // minor is the usual bump
	}
	return nil
}

func (m *Model) resetMenu(hash, label string) *menu {
	mk := func(mode git.ResetMode, title, desc string, danger bool) menuItem {
		return menuItem{label: title, danger: danger, run: func(m *Model) tea.Cmd {
			if mode == git.ResetHard {
				m.confirm("Hard reset "+m.currentBranch()+" to "+label+"?", "Working tree and index are overwritten. Uncommitted changes are lost.", "Reset --hard", true, func(m *Model) tea.Cmd {
					return m.action("Reset --hard", func() error { return m.repo.Reset(mode, hash) })
				})
				return nil
			}
			return m.action("Reset "+string(mode), func() error { return m.repo.Reset(mode, hash) })
		}}
	}
	return &menu{title: "Reset " + m.currentBranch() + " to " + label, items: []menuItem{
		mk(git.ResetSoft, "Soft — keep index and working tree", "", false),
		mk(git.ResetMixed, "Mixed — keep working tree", "", false),
		mk(git.ResetHard, "Hard — discard everything", "", true),
	}}
}

// openBrowser opens a URL with the platform's default handler.
func openBrowser(url string) tea.Cmd {
	return func() tea.Msg {
		var cmd *exec.Cmd
		switch runtime.GOOS {
		case "darwin":
			cmd = exec.Command("open", url)
		case "windows":
			cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
		default:
			cmd = exec.Command("xdg-open", url)
		}
		if err := cmd.Start(); err != nil {
			return clipboardMsg{err: err, what: "url"}
		}
		return nil
	}
}

// ---- menus -------------------------------------------------------------

func (m *Model) menuForCommit(c *git.Commit) *menu {
	hash, short := c.Hash, c.Short
	cur := m.currentBranch()
	items := []menuItem{
		{label: "Checkout revision (detached)", key: "c", run: func(m *Model) tea.Cmd {
			return m.action("Checkout "+short, func() error { return m.repo.CheckoutDetached(hash) })
		}},
		{label: "New branch here…", key: "b", run: func(m *Model) tea.Cmd { return m.newBranchPrompt(hash, short, true) }},
		{label: "New tag here…", key: "t", run: func(m *Model) tea.Cmd { return m.newTagPrompt(hash, short) }},
		sep(),
		{label: "Cherry-pick", key: "p", disabled: c.Hash == m.info.HeadHash, run: func(m *Model) tea.Cmd {
			return m.action("Cherry-pick "+short, func() error { return m.repo.CherryPick(hash) })
		}},
		{label: "Revert commit", key: "v", run: func(m *Model) tea.Cmd {
			return m.action("Revert "+short, func() error { return m.repo.Revert(hash) })
		}},
		{label: "Reset " + cur + " to here", key: "r", sub: func(m *Model) *menu { return m.resetMenu(hash, short) }},
	}
	if c.Hash == m.info.HeadHash && !m.info.Detached {
		items = append(items, menuItem{label: "Undo commit (keep changes)", key: "u", run: func(m *Model) tea.Cmd {
			return m.action("Undo commit", func() error { return m.repo.Reset(git.ResetSoft, "HEAD~1") })
		}})
	}
	// Branch-specific actions for refs on this commit.
	var refItems []menuItem
	for _, r := range c.Refs {
		if r.Kind != git.RefLocal && r.Kind != git.RefRemote {
			continue
		}
		var b *git.Branch
		list := m.refs.Locals
		if r.Kind == git.RefRemote {
			list = m.refs.Remotes
		}
		for i := range list {
			if list[i].Name == r.Name {
				b = &list[i]
			}
		}
		if b == nil || b.IsHead {
			continue
		}
		bb := b
		refItems = append(refItems, menuItem{label: bb.Name, sub: func(m *Model) *menu { return m.branchActions(bb) }})
	}
	if len(refItems) > 0 {
		items = append(items, sep())
		items = append(items, refItems...)
	}
	items = append(items, sep(),
		menuItem{label: "Copy revision number", key: "y", run: func(m *Model) tea.Cmd { return copyToClipboard("hash "+short, hash) }},
		menuItem{label: "Copy subject", run: func(m *Model) tea.Cmd { return copyToClipboard("subject", c.Subject) }},
	)
	if base := remoteWebURL(m.repo); base != "" {
		items = append(items, menuItem{label: "Open in browser", key: "o", run: func(m *Model) tea.Cmd { return openBrowser(base + "/commit/" + hash) }})
	}
	if m.ci != nil {
		if r, ok := m.ciResults[hash]; ok && r.State != ci.StateNone {
			p := m.ci
			items = append(items, menuItem{label: "Open checks in browser", key: "k", run: func(m *Model) tea.Cmd { return openBrowser(p.ChecksURL(hash)) }})
		}
	}
	return &menu{title: short + "  " + trunc(c.Subject, 40), items: items}
}

func (m *Model) menuForLocal() *menu {
	items := []menuItem{
		{label: "Commit…", key: "c", run: func(m *Model) tea.Cmd { return m.openCommit() }},
		{label: "Push…", key: "P", run: func(m *Model) tea.Cmd { return m.openPush() }},
		sep(),
		{label: "Stage all", key: "s", run: func(m *Model) tea.Cmd { return m.action("Stage all", m.repo.StageAll) }},
		{label: "Unstage all", key: "u", run: func(m *Model) tea.Cmd { return m.action("Unstage all", m.repo.UnstageAll) }},
		sep(),
		{label: "Stash changes", key: "S", run: func(m *Model) tea.Cmd { return m.action("Stash", m.repo.StashPush) }},
		{label: "Unstash (pop)", key: "U", run: func(m *Model) tea.Cmd { return m.action("Stash pop", m.repo.StashPop) }},
		sep(),
		{label: "Discard all changes", key: "D", danger: true, run: func(m *Model) tea.Cmd {
			m.confirm("Discard all local changes?", "Every modified file is reset and untracked files are deleted. This cannot be undone.", "Discard all", true, func(m *Model) tea.Cmd {
				return m.action("Discard all", func() error {
					if _, err := m.repo.RunWrite("reset", "-q", "--hard"); err != nil {
						return err
					}
					_, err := m.repo.RunWrite("clean", "-fd")
					return err
				})
			})
			return nil
		}},
	}
	if m.info.State != "" {
		state := m.info.State
		items = append([]menuItem{
			{label: "Continue " + state, run: func(m *Model) tea.Cmd {
				return m.action("Continue", func() error { return m.repo.ContinueState(state) })
			}},
			{label: "Abort " + state, danger: true, run: func(m *Model) tea.Cmd { return m.action("Abort", func() error { return m.repo.AbortState(state) }) }},
			sep(),
		}, items...)
	}
	return &menu{title: "Uncommitted changes", items: items}
}

func (m *Model) branchActions(b *git.Branch) *menu {
	cur := m.currentBranch()
	name := b.Name
	items := []menuItem{
		{label: "Show in log", key: "s", run: func(m *Model) tea.Cmd { return m.showRefInLog(name) }},
		{label: "Checkout", key: "c", disabled: b.IsHead, run: func(m *Model) tea.Cmd { return m.checkoutBranch(b) }},
		sep(),
		{label: "Merge into " + cur, key: "m", disabled: b.IsHead, run: func(m *Model) tea.Cmd {
			return m.action("Merge "+name, func() error { return m.repo.Merge(name) })
		}},
		{label: "Rebase " + cur + " onto " + trunc(name, 24), key: "r", disabled: b.IsHead, run: func(m *Model) tea.Cmd {
			return m.action("Rebase onto "+name, func() error { return m.repo.Rebase(name) })
		}},
		sep(),
		{label: "New branch from here…", key: "b", run: func(m *Model) tea.Cmd { return m.newBranchPrompt(name, name, true) }},
		{label: "New tag here…", key: "t", run: func(m *Model) tea.Cmd { return m.newTagPrompt(name, name) }},
	}
	switch b.Kind {
	case git.RefLocal:
		items = append(items, sep(),
			menuItem{label: "Rename…", key: "R", run: func(m *Model) tea.Cmd {
				return m.prompt("Rename branch", "Rename "+name+" to:", "new name", name, func(m *Model, nn string) tea.Cmd {
					return m.action("Rename", func() error { return m.repo.RenameBranch(name, nn) })
				})
			}},
		)
		if b.IsHead {
			items = append(items, menuItem{label: "Push…", key: "P", run: func(m *Model) tea.Cmd { return m.openPush() }},
				menuItem{label: "Pull…", key: "p", run: func(m *Model) tea.Cmd { return m.pullChooser() }})
		} else {
			items = append(items, menuItem{label: "Push", key: "P", run: func(m *Model) tea.Cmd {
				return m.action("Push "+name, func() error { _, err := m.repo.RunWrite("push", "-u", "origin", name); return err })
			}})
		}
		items = append(items, sep(), menuItem{label: "Delete", key: "d", danger: true, disabled: b.IsHead, run: func(m *Model) tea.Cmd { return m.deleteRef(b) }})
	case git.RefRemote:
		items = append(items, sep(),
			menuItem{label: "Pull into " + cur, key: "p", run: func(m *Model) tea.Cmd {
				return m.action("Pull "+name, func() error {
					_, err := m.repo.RunWrite("pull", b.Remote, strings.TrimPrefix(name, b.Remote+"/"))
					return err
				})
			}},
			menuItem{label: "Delete on " + b.Remote, key: "d", danger: true, run: func(m *Model) tea.Cmd { return m.deleteRef(b) }},
		)
	case git.RefTag:
		items = append(items, sep())
		if remote := m.defaultRemote(); remote != "" {
			items = append(items,
				menuItem{label: "Push tag to " + remote, key: "P", run: func(m *Model) tea.Cmd {
					return m.action("Push "+name, func() error { return m.repo.PushTag(remote, name) })
				}},
				menuItem{label: "Delete tag on " + remote, danger: true, run: func(m *Model) tea.Cmd {
					m.confirm("Delete "+name+" on "+remote+"?", "The tag is removed from the remote for everyone.", "Delete remote tag", true, func(m *Model) tea.Cmd {
						return m.action("Delete "+name+" on "+remote, func() error { return m.repo.DeleteRemoteTag(remote, name) })
					})
					return nil
				}},
			)
		}
		items = append(items, menuItem{label: "Delete local tag", key: "d", danger: true, run: func(m *Model) tea.Cmd { return m.deleteRef(b) }})
	}
	items = append(items, sep(), menuItem{label: "Copy name", key: "y", run: func(m *Model) tea.Cmd { return copyToClipboard("name", name) }})
	return &menu{title: name, items: items}
}

func (m *Model) menuForBranch(n *bnode) *menu {
	if n.kind == bHead {
		items := []menuItem{
			{label: "Show current branch in log", key: "s", run: func(m *Model) tea.Cmd { m.logOpts.All = false; m.logOpts.Ref = ""; return m.applyFilter() }},
			{label: "Show all branches", key: "A", run: func(m *Model) tea.Cmd { return m.showRefInLog("") }},
			sep(),
			{label: "Commit…", key: "c", run: func(m *Model) tea.Cmd { return m.openCommit() }},
			{label: "Push…", key: "P", run: func(m *Model) tea.Cmd { return m.openPush() }},
			{label: "Pull…", key: "p", run: func(m *Model) tea.Cmd { return m.pullChooser() }},
			{label: "Fetch (all, prune)", key: "f", run: func(m *Model) tea.Cmd { return m.action("Fetch", m.repo.Fetch) }},
			sep(),
			{label: "New branch…", key: "b", run: func(m *Model) tea.Cmd { return m.newBranchPrompt("HEAD", "HEAD", true) }},
			{label: "New version tag…", key: "v", run: func(m *Model) tea.Cmd { return m.versionTagMenu() }},
		}
		return &menu{title: n.label, items: items}
	}
	if n.branch == nil {
		return nil
	}
	return m.branchActions(n.branch)
}

func (m *Model) menuForFile(n *fnode) *menu {
	if n.isDir {
		return &menu{title: n.label, items: []menuItem{
			{label: "Fold / unfold", key: " ", run: func(m *Model) tea.Cmd { return m.filesKey(" ") }},
			{label: "Copy path", key: "y", run: func(m *Model) tea.Cmd { return copyToClipboard("path", strings.TrimPrefix(n.key, "/")) }},
		}}
	}
	f := *n.file
	items := []menuItem{
		{label: "Show diff", key: "enter", run: func(m *Model) tea.Cmd { return m.openDiff(f) }},
		{label: "Show history in log", key: "H", run: func(m *Model) tea.Cmd { return m.showPathHistory(f.Path) }},
		{label: "Copy path", key: "y", run: func(m *Model) tea.Cmd { return copyToClipboard("path", f.Path) }},
	}
	if m.filesFor == "local" {
		check := "Check for commit"
		if m.selected[f.Path] {
			check = "Uncheck"
		}
		stage := "Stage (git add)"
		if f.Staged && !f.Unstaged {
			stage = "Unstage (git reset)"
		}
		items = append(items, sep(),
			menuItem{label: check, key: " ", run: func(m *Model) tea.Cmd { m.selected[f.Path] = !m.selected[f.Path]; return nil }},
			menuItem{label: "Commit…", key: "c", run: func(m *Model) tea.Cmd { return m.openCommit() }},
			sep(),
			menuItem{label: stage, key: "s", run: func(m *Model) tea.Cmd { return m.toggleStage(f) }},
			menuItem{label: "Discard changes", key: "d", danger: true, run: func(m *Model) tea.Cmd { return m.discardPrompt(f) }},
		)
	}
	return &menu{title: n.label, items: items}
}
