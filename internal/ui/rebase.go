package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/jinhyo-dev/gitpad/internal/git"
	"github.com/jinhyo-dev/gitpad/internal/ui/theme"
)

// The rebase workspace replaces the log pane with an editable todo list for
// `git rebase -i`: every commit from the chosen one up to HEAD, newest first
// like the log, with an action per row and reordering.

type rebaseState struct {
	plan   git.RebasePlan
	steps  []git.RebaseStep // display order: newest first
	cur    int
	scroll int
}

type rebasePlanMsg struct {
	plan git.RebasePlan
	err  error
}

// startRebase plans a rebase from commit c (inclusive) to HEAD.
func (m *Model) startRebase(c *git.Commit) tea.Cmd {
	if c == nil {
		return nil
	}
	if m.info.State != "" {
		return m.showToast("Finish or abort the current "+m.info.State+" first", 2)
	}
	m.loading++
	repo := m.repo
	hash := c.Hash
	return func() tea.Msg {
		plan, err := repo.PlanRebase(hash)
		return rebasePlanMsg{plan: plan, err: err}
	}
}

func (m *Model) onRebasePlan(msg rebasePlanMsg) tea.Cmd {
	m.loading--
	if msg.err != nil {
		return m.showToast("Rebase: "+msg.err.Error(), 2)
	}
	steps := make([]git.RebaseStep, 0, len(msg.plan.Commits))
	for _, c := range msg.plan.Commits {
		steps = append(steps, git.RebaseStep{Hash: c.Hash, Subject: c.Subject, Action: git.Pick})
	}
	m.rebase = &rebaseState{plan: msg.plan, steps: steps}
	m.rebaseOpen = true
	m.diff, m.console = nil, false
	m.focus = PanelLog
	return m.rebaseSyncSelection()
}

func (m *Model) closeRebase() {
	m.rebaseOpen = false
	m.rebase = nil
}

// rebaseSyncSelection points the log cursor at the highlighted commit so the
// Changes / Details panes follow along.
func (m *Model) rebaseSyncSelection() tea.Cmd {
	r := m.rebase
	if r == nil || r.cur < 0 || r.cur >= len(r.steps) {
		return nil
	}
	if row := m.rowOfHash(r.steps[r.cur].Hash); row >= 0 {
		m.lcur = row
		return m.scheduleSelection()
	}
	return nil
}

func (m *Model) rebaseTitle() string {
	r := m.rebase
	onto := "root"
	if !r.plan.Root {
		onto = r.plan.Base[:minInt(8, len(r.plan.Base))]
	}
	return fmt.Sprintf("Interactive rebase · %s onto %s", plural(len(r.steps), "commit", "commits"), onto)
}

var actionStyles = map[git.RebaseAction]lipgloss.Style{
	git.Pick:   lipgloss.NewStyle().Foreground(theme.Text),
	git.Reword: lipgloss.NewStyle().Foreground(theme.Yellow).Bold(true),
	git.Edit:   lipgloss.NewStyle().Foreground(theme.Orange).Bold(true),
	git.Squash: lipgloss.NewStyle().Foreground(theme.Magenta).Bold(true),
	git.Fixup:  lipgloss.NewStyle().Foreground(theme.Magenta),
	git.Drop:   lipgloss.NewStyle().Foreground(theme.Red).Bold(true),
}

func (m *Model) renderRebase(w, h int) []string {
	r := m.rebase
	listH := h - 3 // legend + separator + button row
	ensureVisible(r.cur, &r.scroll, listH)
	var out []string
	end := minInt(len(r.steps), r.scroll+listH)
	for i := r.scroll; i < end; i++ {
		s := r.steps[i]
		st := actionStyles[s.Action]
		row := " " + st.Render(pad(string(s.Action), 6)) + " " + lipgloss.NewStyle().Foreground(theme.Accent).Render(s.Hash[:8]) + "  "
		subject := s.Subject
		subjSt := theme.Base
		switch s.Action {
		case git.Drop:
			subjSt = theme.DimSt.Strikethrough(true)
		case git.Squash, git.Fixup:
			subjSt = theme.MutedSt
		}
		rest := w - width(row)
		if s.Action == git.Reword && s.Message != "" {
			nm := lipgloss.NewStyle().Foreground(theme.Yellow).Render("→ " + trunc(strings.SplitN(s.Message, "\n", 2)[0], rest/2))
			row += subjSt.Render(trunc(subject, rest-width(nm)-2)) + "  " + nm
		} else {
			row += subjSt.Render(trunc(subject, rest))
		}
		if i == r.cur {
			row = highlight(pad(row, w), m.selectionBg(PanelLog))
		}
		out = append(out, pad(row, w))
	}
	for len(out) < listH {
		out = append(out, strings.Repeat(" ", w))
	}
	out = append(out, separator(w, "", ""))
	legend := keyHints("p", "pick", "r", "reword", "e", "edit", "s", "squash", "f", "fixup", "d", "drop", "⇧↑↓", "move")
	out = append(out, pad(" "+legend, w))
	btn := " " + theme.ButtonPrimary.Render("Start rebase") + "  " + theme.ButtonPlain.Render("Cancel")
	hint := theme.KeyHint.Render(keyLabel("ctrl+s")) + theme.KeyLabel.Render(" start  ") + theme.KeyHint.Render("esc") + theme.KeyLabel.Render(" cancel ")
	out = append(out, joinRow(btn, hint, w))
	return out
}

func (m *Model) rebaseKey(k tea.KeyMsg) tea.Cmd {
	r := m.rebase
	key := k.String()
	set := func(a git.RebaseAction) tea.Cmd {
		if r.cur < len(r.steps) {
			r.steps[r.cur].Action = a
		}
		return nil
	}
	switch key {
	case "esc", "q":
		m.closeRebase()
		return nil
	case "ctrl+s":
		return m.confirmRebase()
	case "j", "down":
		r.cur = minInt(r.cur+1, len(r.steps)-1)
		return m.rebaseSyncSelection()
	case "k", "up":
		r.cur = maxInt(r.cur-1, 0)
		return m.rebaseSyncSelection()
	case "g", "home":
		r.cur = 0
		return m.rebaseSyncSelection()
	case "G", "end":
		r.cur = len(r.steps) - 1
		return m.rebaseSyncSelection()
	case "shift+up", "K":
		if r.cur > 0 {
			r.steps[r.cur], r.steps[r.cur-1] = r.steps[r.cur-1], r.steps[r.cur]
			r.cur--
		}
		return nil
	case "shift+down", "J":
		if r.cur < len(r.steps)-1 {
			r.steps[r.cur], r.steps[r.cur+1] = r.steps[r.cur+1], r.steps[r.cur]
			r.cur++
		}
		return nil
	case "p":
		return set(git.Pick)
	case "e":
		return set(git.Edit)
	case "s":
		return set(git.Squash)
	case "f":
		return set(git.Fixup)
	case "d":
		return set(git.Drop)
	case "r":
		return m.rewordPrompt()
	case "enter", "m":
		x, y := m.menuAnchorFor(PanelLog, r.cur)
		m.openMenu(m.rebaseRowMenu(), x, y)
		return nil
	}
	return nil
}

func (m *Model) rewordPrompt() tea.Cmd {
	r := m.rebase
	if r.cur >= len(r.steps) {
		return nil
	}
	s := &r.steps[r.cur]
	initial := s.Message
	if initial == "" {
		initial = s.Subject
	}
	return m.prompt("Reword "+s.Hash[:8], "New commit message (first line becomes the subject).", "commit message", initial, func(m *Model, msg string) tea.Cmd {
		if m.rebase != nil && r.cur < len(m.rebase.steps) {
			m.rebase.steps[r.cur].Action = git.Reword
			m.rebase.steps[r.cur].Message = msg
		}
		return nil
	})
}

func (m *Model) rebaseRowMenu() *menu {
	r := m.rebase
	s := r.steps[r.cur]
	mk := func(label string, key string, a git.RebaseAction) menuItem {
		return menuItem{label: label, key: key, danger: a == git.Drop, run: func(m *Model) tea.Cmd {
			if m.rebase != nil && r.cur < len(m.rebase.steps) {
				m.rebase.steps[r.cur].Action = a
			}
			return nil
		}}
	}
	return &menu{title: s.Hash[:8] + "  " + trunc(s.Subject, 40), items: []menuItem{
		mk("Pick — keep as is", "p", git.Pick),
		{label: "Reword — change the message…", key: "r", run: func(m *Model) tea.Cmd { return m.rewordPrompt() }},
		mk("Edit — stop here to amend", "e", git.Edit),
		mk("Squash — merge into the previous commit, keep both messages", "s", git.Squash),
		mk("Fixup — merge into the previous commit, discard this message", "f", git.Fixup),
		sep(),
		mk("Drop — remove this commit", "d", git.Drop),
	}}
}

// confirmRebase validates the plan and asks before rewriting history.
func (m *Model) confirmRebase() tea.Cmd {
	r := m.rebase
	steps := m.rebaseExecutionOrder()
	if len(steps) == 0 {
		return m.showToast("Nothing to rebase", 2)
	}
	if a := steps[0].Action; a == git.Squash || a == git.Fixup {
		return m.showToast("The oldest commit cannot be squashed — there is nothing before it", 2)
	}
	drops, changes := 0, 0
	for _, s := range steps {
		if s.Action == git.Drop {
			drops++
		}
		if s.Action != git.Pick {
			changes++
		}
	}
	for i, s := range steps {
		if s.Hash != r.plan.Commits[len(r.plan.Commits)-1-i].Hash {
			changes++
			break
		}
	}
	if changes == 0 {
		return m.showToast("The plan does not change anything", 0)
	}
	body := fmt.Sprintf("%s will be rewritten on %s.", plural(len(steps), "commit", "commits"), m.currentBranch())
	if drops > 0 {
		body += fmt.Sprintf(" %s dropped.", plural(drops, "commit is", "commits are"))
	}
	body += " The previous history stays reachable as ORIG_HEAD."
	m.confirm("Rebase "+m.currentBranch()+"?", body, "Start rebase", drops > 0, func(m *Model) tea.Cmd {
		plan := r.plan
		repo := m.repo
		m.closeRebase()
		return m.actionWith("Rebase", func() error { return repo.RebaseInteractive(plan, steps) }, func(m *Model, err error) tea.Cmd {
			return m.showToast("Rebase stopped: "+err.Error()+" — resolve, then Continue / Abort from the changes menu", 2)
		})
	})
	return nil
}

// rebaseExecutionOrder returns the steps oldest-first as git expects.
func (m *Model) rebaseExecutionOrder() []git.RebaseStep {
	r := m.rebase
	out := make([]git.RebaseStep, 0, len(r.steps))
	for i := len(r.steps) - 1; i >= 0; i-- {
		out = append(out, r.steps[i])
	}
	return out
}

// rebaseMouse selects rows and opens the row menu on right-click.
func (m *Model) rebaseMouse(msg tea.MouseMsg) tea.Cmd {
	r := m.rebase
	rect := m.rects[PanelLog]
	if !rect.contains(msg.X, msg.Y) {
		return nil
	}
	if msg.Button == tea.MouseButtonWheelUp || msg.Button == tea.MouseButtonWheelDown {
		d := 3
		if msg.Button == tea.MouseButtonWheelUp {
			d = -3
		}
		r.cur = clamp(r.cur+d, 0, len(r.steps)-1)
		return m.rebaseSyncSelection()
	}
	if msg.Action != tea.MouseActionPress {
		return nil
	}
	row := r.scroll + msg.Y - rect.y
	if row < 0 || row >= len(r.steps) {
		return nil
	}
	r.cur = row
	cmd := m.rebaseSyncSelection()
	if msg.Button == tea.MouseButtonRight {
		m.menuAnchor = &[2]int{msg.X, msg.Y}
		m.openMenu(m.rebaseRowMenu(), msg.X, msg.Y)
		m.menuAnchor = nil
	}
	return cmd
}
