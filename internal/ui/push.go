package ui

import (
	"fmt"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/jinhyo-dev/gitpad/internal/git"
	"github.com/jinhyo-dev/gitpad/internal/ui/theme"
)

// pushState backs the Push dialog: what will be sent, and to where.
type pushState struct {
	loading     bool
	err         string
	branch      string
	remote      string
	upstream    string
	newUpstream bool
	noRemote    bool
	commits     []git.Commit
	behind      int
	force       bool
	tags        bool
	cur         int // 0 force, 1 tags, 2 push
	busy        bool
}

type pushDataMsg struct {
	branch, remote, upstream string
	newUpstream, noRemote    bool
	detached                 bool
	commits                  []git.Commit
	behind                   int
}

const pushMaxCommits = 8

func (m *Model) openPush() tea.Cmd {
	m.push = &pushState{loading: true, cur: 2}
	repo := m.repo
	return func() tea.Msg {
		msg := pushDataMsg{}
		out, err := repo.Run("symbolic-ref", "--short", "-q", "HEAD")
		if err != nil || strings.TrimSpace(out) == "" {
			msg.detached = true
			return msg
		}
		msg.branch = strings.TrimSpace(out)
		remotes := repo.Remotes()
		if len(remotes) == 0 {
			msg.noRemote = true
			return msg
		}
		if out, err := repo.Run("rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{u}"); err == nil && strings.TrimSpace(out) != "" {
			msg.upstream = strings.TrimSpace(out)
			msg.remote = strings.SplitN(msg.upstream, "/", 2)[0]
			msg.commits, _ = repo.Log(git.LogOptions{Extra: []string{"@{u}..HEAD"}, Limit: 200})
			if out, err := repo.Run("rev-list", "--count", "HEAD..@{u}"); err == nil {
				msg.behind, _ = strconv.Atoi(strings.TrimSpace(out))
			}
		} else {
			msg.newUpstream = true
			msg.remote = remotes[0]
			for _, r := range remotes {
				if r == "origin" {
					msg.remote = r
				}
			}
			msg.commits, _ = repo.Log(git.LogOptions{Extra: []string{"HEAD", "--not", "--remotes"}, Limit: 200})
		}
		return msg
	}
}

func (m *Model) onPushData(msg pushDataMsg) {
	p := m.push
	if p == nil {
		return
	}
	p.loading = false
	switch {
	case msg.detached:
		p.err = "HEAD is detached — checkout a branch before pushing."
	case msg.noRemote:
		p.err = "No remote configured. Add one with: git remote add origin <url>"
	}
	p.branch, p.remote, p.upstream = msg.branch, msg.remote, msg.upstream
	p.newUpstream, p.noRemote = msg.newUpstream, msg.noRemote
	p.commits, p.behind = msg.commits, msg.behind
}

func (m *Model) pushKey(k tea.KeyMsg) tea.Cmd {
	p := m.push
	switch k.String() {
	case "esc", "q":
		if !p.busy {
			m.push = nil
		}
	case "f":
		p.force = !p.force
	case "t":
		p.tags = !p.tags
	case "j", "down", "tab":
		p.cur = (p.cur + 1) % 3
	case "k", "up", "shift+tab":
		p.cur = (p.cur + 2) % 3
	case " ":
		switch p.cur {
		case 0:
			p.force = !p.force
		case 1:
			p.tags = !p.tags
		default:
			return m.doPush()
		}
	case "enter":
		return m.doPush()
	}
	return nil
}

func (m *Model) doPush() tea.Cmd {
	p := m.push
	if p == nil || p.loading || p.busy || p.err != "" {
		return nil
	}
	if p.behind > 0 && !p.force {
		m.confirm(fmt.Sprintf("%s is %d commits behind %s", p.branch, p.behind, p.upstream),
			"The push will be rejected unless you pull first. Push anyway?", "Push anyway", false, func(m *Model) tea.Cmd {
				return m.runPush()
			})
		return nil
	}
	return m.runPush()
}

func (m *Model) runPush() tea.Cmd {
	p := m.push
	p.busy = true
	opts := git.PushOptions{Remote: p.remote, Branch: p.branch, SetUpstream: p.newUpstream, Force: p.force, Tags: p.tags}
	repo := m.repo
	label := "Push " + p.branch
	if p.force {
		label = "Force push " + p.branch
	}
	return m.actionThen(label, func() error { return repo.PushWith(opts) }, func(m *Model) tea.Cmd {
		m.push = nil
		return nil
	})
}

// pushBox returns the dialog geometry (deterministic, for View and mouse).
func (m *Model) pushBox() (x, y, w, h int) {
	w = minInt(70, m.width-4)
	body := m.pushLines(w - 6)
	h = len(body) + 4
	x = (m.width - w) / 2
	y = (m.height - h) / 2
	return
}

func (m *Model) pushLines(w int) []string {
	p := m.push
	var lines []string
	add := func(s string) { lines = append(lines, pad(s, w)) }
	if p.loading {
		add(m.spin.View() + theme.MutedSt.Render(" collecting commits…"))
		return lines
	}
	if p.err != "" {
		add(theme.MenuDanger.Render(trunc(p.err, w)))
		return lines
	}
	target := p.upstream
	if p.newUpstream {
		target = p.remote + "/" + p.branch + theme.DimSt.Render("  (new upstream)")
	}
	add(theme.AccentB.Render(p.branch) + theme.MutedSt.Render("  →  ") + lipgloss.NewStyle().Foreground(theme.Magenta).Render(target))
	add("")
	switch n := len(p.commits); {
	case n == 0:
		add(theme.MutedSt.Render("Nothing to push — " + p.branch + " is up to date"))
	default:
		add(theme.Bold.Render(plural(n, "commit", "commits") + " will be pushed"))
		for i, c := range p.commits {
			if i >= pushMaxCommits {
				add(theme.DimSt.Render(fmt.Sprintf("  … and %d more", n-pushMaxCommits)))
				break
			}
			add("  " + lipgloss.NewStyle().Foreground(theme.Accent).Render(c.Short) + "  " + theme.Base.Render(trunc(c.Subject, w-12)))
		}
	}
	if p.behind > 0 {
		add("")
		add(lipgloss.NewStyle().Foreground(theme.Orange).Render(fmt.Sprintf("⚠ %s is %d commits behind %s — pull first, or push will be rejected", p.branch, p.behind, p.upstream)))
	}
	add("")
	check := func(on bool, label string, focused bool) string {
		box := theme.DimSt.Render("[ ]")
		if on {
			box = lipgloss.NewStyle().Foreground(theme.Accent).Bold(true).Render("[x]")
		}
		st := theme.Base
		if focused {
			st = theme.Bold
		}
		s := box + " " + st.Render(label)
		if focused {
			return highlight(s, theme.Selection)
		}
		return s
	}
	add(check(p.force, "Force with lease  (f)", p.cur == 0) + "   " + check(p.tags, "Push tags  (t)", p.cur == 1))
	add("")
	btn := theme.ButtonPlain
	if p.cur == 2 {
		btn = theme.ButtonPrimary
	}
	label := "Push"
	if p.force {
		label = "Force push"
		if p.cur == 2 {
			btn = theme.ButtonDanger
		}
	}
	if p.busy {
		label = "Pushing…"
	}
	hint := theme.DimSt.Render("enter push · esc cancel")
	buttons := btn.Render(label) + " " + theme.ButtonPlain.Render("Cancel")
	add(joinRow(buttons, hint, w))
	return lines
}

func (m *Model) renderPush() string {
	_, _, w, _ := m.pushBox()
	inner := w - 6
	lines := append([]string{pad(theme.DialogTitle.Render("Push"), inner), ""}, m.pushLines(inner)...)
	box := theme.DialogBox
	if m.push != nil && m.push.force {
		box = theme.DialogBoxDanger
	}
	return box.Render(strings.Join(lines, "\n"))
}

// pushMouse maps a click to the checkbox / button rows.
func (m *Model) pushMouse(msg tea.MouseMsg) tea.Cmd {
	if msg.Action != tea.MouseActionPress || msg.Button != tea.MouseButtonLeft {
		return nil
	}
	x, y, w, h := m.pushBox()
	if msg.X < x || msg.X >= x+w || msg.Y < y || msg.Y >= y+h {
		if !m.push.busy {
			m.push = nil
		}
		return nil
	}
	body := m.pushLines(w - 6)
	row := msg.Y - y - 4 // border + padding + title + blank
	if row < 0 || row >= len(body) {
		return nil
	}
	p := m.push
	rel := msg.X - x - 3
	switch row {
	case len(body) - 3: // checkboxes
		if rel < width("[x] Force with lease  (f)")+2 {
			p.force = !p.force
			p.cur = 0
		} else {
			p.tags = !p.tags
			p.cur = 1
		}
	case len(body) - 1: // buttons
		if rel < width(theme.ButtonPrimary.Render("Force push")) {
			p.cur = 2
			return m.doPush()
		}
		if !p.busy {
			m.push = nil
		}
	}
	return nil
}

// pullChooser offers merge / rebase / fetch-only.
func (m *Model) pullChooser() tea.Cmd {
	if m.lastPull == "" {
		if m.repo.PullRebaseDefault() {
			m.lastPull = "rebase"
		} else {
			m.lastPull = "merge"
		}
	}
	title := "Pull"
	if m.info.Upstream != "" {
		title = "Pull " + m.info.Upstream + " → " + m.currentBranch()
	}
	mn := &menu{title: title, items: []menuItem{
		{label: "Pull (merge)", key: "m", run: func(m *Model) tea.Cmd { m.lastPull = "merge"; return m.action("Pull", m.repo.PullMerge) }},
		{label: "Pull (rebase)", key: "r", run: func(m *Model) tea.Cmd { m.lastPull = "rebase"; return m.action("Pull --rebase", m.repo.PullRebase) }},
		{label: "Fetch only", key: "f", run: func(m *Model) tea.Cmd { m.lastPull = "fetch"; return m.action("Fetch", m.repo.Fetch) }},
	}}
	mn.layout()
	m.openMenu(mn, (m.width-mn.w)/2, (m.height-mn.h)/2)
	for i, it := range mn.items {
		if (m.lastPull == "merge" && it.key == "m") || (m.lastPull == "rebase" && it.key == "r") || (m.lastPull == "fetch" && it.key == "f") {
			mn.cur = i
		}
	}
	return nil
}
