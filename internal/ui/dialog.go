package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/jinhyo-dev/gitpad/internal/ui/theme"
)

type dialogKind int

const (
	dlgConfirm dialogKind = iota
	dlgInput
)

type dialog struct {
	kind       dialogKind
	title      string
	body       string
	ok         string
	danger     bool
	quit       bool // the quit confirmation: a second `q` confirms
	allowEmpty bool // input dialogs: Enter with no text is accepted
	input      textinput.Model
	onOK       func(m *Model, value string) tea.Cmd
}

// confirmQuit asks before leaving; `q` again, `y` or Enter exits.
func (m *Model) confirmQuit() {
	m.dialog = &dialog{kind: dlgConfirm, title: "Quit gitpad?", body: "Press q again or Enter to exit, Esc to stay.", ok: "Quit", quit: true,
		onOK: func(m *Model, _ string) tea.Cmd { return tea.Quit }}
}

func (m *Model) confirm(title, body, ok string, danger bool, onOK func(m *Model) tea.Cmd) {
	m.dialog = &dialog{kind: dlgConfirm, title: title, body: body, ok: ok, danger: danger,
		onOK: func(m *Model, _ string) tea.Cmd { return onOK(m) }}
}

// promptOptional is prompt() where an empty answer is a valid answer.
func (m *Model) promptOptional(title, body, placeholder string, onOK func(m *Model, value string) tea.Cmd) tea.Cmd {
	cmd := m.prompt(title, body, placeholder, "", onOK)
	m.dialog.allowEmpty = true
	m.dialog.ok = "Continue"
	return cmd
}

func (m *Model) prompt(title, body, placeholder, initial string, onOK func(m *Model, value string) tea.Cmd) tea.Cmd {
	ti := textinput.New()
	ti.Prompt = "› "
	ti.PromptStyle = lipgloss.NewStyle().Foreground(theme.Accent)
	ti.Placeholder = placeholder
	ti.PlaceholderStyle = lipgloss.NewStyle().Foreground(theme.Dim)
	ti.TextStyle = lipgloss.NewStyle().Foreground(theme.Text)
	ti.Cursor.Style = lipgloss.NewStyle().Foreground(theme.Accent)
	ti.CharLimit = 200
	ti.Width = 44
	ti.SetValue(initial)
	ti.CursorEnd()
	cmd := ti.Focus() // focus before copying into the dialog
	m.dialog = &dialog{kind: dlgInput, title: title, body: body, ok: "OK", input: ti, onOK: onOK}
	return cmd
}

func (d *dialog) render() string {
	w := 52
	var lines []string
	lines = append(lines, theme.DialogTitle.Render(d.title))
	if d.body != "" {
		lines = append(lines, "")
		for _, l := range strings.Split(ansi.Wordwrap(d.body, w, ""), "\n") {
			lines = append(lines, theme.DialogBody.Render(l))
		}
	}
	lines = append(lines, "")
	if d.kind == dlgInput {
		lines = append(lines, d.input.View())
		lines = append(lines, "")
	}
	okBtn := theme.ButtonPrimary
	if d.danger {
		okBtn = theme.ButtonDanger
	}
	hint := theme.DimSt.Render("enter ↵ · esc")
	if d.quit {
		hint = theme.DimSt.Render("q / enter · esc")
	}
	buttons := okBtn.Render(d.ok) + " " + theme.ButtonPlain.Render("Cancel")
	gap := w - width(buttons) - width(hint)
	if gap < 1 {
		gap = 1
	}
	lines = append(lines, buttons+strings.Repeat(" ", gap)+hint)
	for i := range lines {
		lines[i] = pad(lines[i], w)
	}
	box := theme.DialogBox
	if d.danger {
		box = theme.DialogBoxDanger
	}
	return box.Render(strings.Join(lines, "\n"))
}
