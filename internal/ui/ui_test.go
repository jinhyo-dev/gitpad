package ui

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

func run(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=Tester", "GIT_AUTHOR_EMAIL=t@example.com",
		"GIT_COMMITTER_NAME=Tester", "GIT_COMMITTER_EMAIL=t@example.com",
		"GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func write(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// makeRepo builds a small repository with a merge, a tag and dirty files.
func makeRepo(t *testing.T) string {
	dir := t.TempDir()
	run(t, dir, "init", "-q", "-b", "main")
	write(t, dir, "README.md", "# demo\n")
	run(t, dir, "add", ".")
	run(t, dir, "commit", "-q", "-m", "chore: 초기 커밋")
	write(t, dir, "main.go", "package main\n\nfunc main() {}\n")
	run(t, dir, "add", ".")
	run(t, dir, "commit", "-q", "-m", "feat: add main")
	run(t, dir, "checkout", "-q", "-b", "feat/side")
	os.MkdirAll(filepath.Join(dir, "pkg", "util"), 0o755)
	write(t, dir, "pkg/util/x.go", "package util\n\nvar X = 1\n")
	run(t, dir, "add", ".")
	run(t, dir, "commit", "-q", "-m", "feat: 유틸 추가 (side branch)")
	run(t, dir, "checkout", "-q", "main")
	write(t, dir, "main.go", "package main\n\nimport \"fmt\"\n\nfunc main() { fmt.Println(1) }\n")
	run(t, dir, "commit", "-q", "-am", "fix: print")
	run(t, dir, "merge", "-q", "--no-ff", "-m", "Merge branch 'feat/side'", "feat/side")
	run(t, dir, "tag", "v0.1.0")
	write(t, dir, "main.go", "package main\n\nimport \"fmt\"\n\nfunc main() { fmt.Println(2) }\n")
	write(t, dir, "new.txt", "hello\n")
	return dir
}

func keyMsg(s string) tea.KeyMsg {
	switch s {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEscape}
	case "tab":
		return tea.KeyMsg{Type: tea.KeyTab}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case " ":
		return tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}}
	case "ctrl+d":
		return tea.KeyMsg{Type: tea.KeyCtrlD}
	case "ctrl+s":
		return tea.KeyMsg{Type: tea.KeyCtrlS}
	case "ctrl+p":
		return tea.KeyMsg{Type: tea.KeyCtrlP}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

type harness struct {
	t     *testing.T
	model tea.Model
	w, h  int
}

func newHarness(t *testing.T, dir string, w, h int) *harness {
	toastDuration, toastErrDuration, watchInterval, selectDebounce, ciRefreshInterval = time.Millisecond, time.Millisecond, time.Millisecond, 0, time.Millisecond
	m := New(dir)
	if m.fatal != nil {
		t.Fatal(m.fatal)
	}
	var model tea.Model = m
	model = drain(model, tea.WindowSizeMsg{Width: w, Height: h}, 0)
	model = drain(model, m.Init()(), 0) // goes through initMsg like the real program
	hs := &harness{t: t, model: model, w: w, h: h}
	hs.check("initial")
	return hs
}

func (h *harness) m() Model { return h.model.(Model) }

func (h *harness) press(keys ...string) {
	for _, k := range keys {
		h.model = drain(h.model, keyMsg(k), 0)
		h.check("after key " + k)
	}
}

func (h *harness) mouse(x, y int, btn tea.MouseButton) {
	h.model = drain(h.model, tea.MouseMsg{X: x, Y: y, Button: btn, Action: tea.MouseActionPress}, 0)
	h.check("after mouse")
}

// check enforces the layout invariant: h lines, each exactly w cells wide.
func (h *harness) check(when string) {
	h.t.Helper()
	view := h.model.View()
	lines := strings.Split(view, "\n")
	if len(lines) != h.h {
		h.t.Fatalf("%s: expected %d lines, got %d", when, h.h, len(lines))
	}
	for i, l := range lines {
		if lw := ansi.StringWidth(l); lw != h.w {
			h.t.Fatalf("%s: line %d width %d != %d:\n%q", when, i, lw, h.w, ansi.Strip(l))
		}
	}
}

func TestWalkthrough(t *testing.T) {
	toastDuration, toastErrDuration, watchInterval = time.Millisecond, time.Millisecond, time.Millisecond
	dir := makeRepo(t)
	h := newHarness(t, dir, 140, 36)
	m := h.m()
	if len(m.commits) != 5 {
		t.Fatalf("expected 5 commits, got %d", len(m.commits))
	}
	if !m.hasLocalRow() {
		t.Fatal("expected uncommitted-changes row")
	}
	if m.filesFor != "local" || len(m.files) != 2 {
		t.Fatalf("expected 2 local files, got %q %d", m.filesFor, len(m.files))
	}
	view := ansi.Strip(h.model.View())
	for _, want := range []string{"Uncommitted changes", "Merge branch 'feat/side'", "v0.1.0", "feat/side", "◉ main"} {
		if !strings.Contains(view, want) {
			t.Errorf("view missing %q", want)
		}
	}

	// Move onto the merge commit and open its actions.
	h.press("j")
	m = h.m()
	if c := m.selectedCommit(); c == nil || !c.IsMerge() {
		t.Fatal("expected merge commit selected")
	}
	if m.filesFor != m.selectedCommit().Hash {
		t.Fatalf("files not loaded for selection: %q", m.filesFor)
	}
	h.press("enter")
	if h.m().menu == nil {
		t.Fatal("menu should be open")
	}
	h.press("j", "j", "esc")
	if h.m().menu != nil {
		t.Fatal("menu should be closed")
	}

	// Search by text, then clear.
	h.press("/")
	if !h.m().searching {
		t.Fatal("search should be active")
	}
	h.press("f", "e", "a", "t", "enter")
	m = h.m()
	if m.logOpts.Grep != "feat" || len(m.commits) != 3 {
		t.Fatalf("grep filter failed: %q %d", m.logOpts.Grep, len(m.commits))
	}
	h.press("esc")
	if mm := h.m(); (&mm).hasFilter() {
		t.Fatal("filters should be cleared")
	}

	// The same box matches author names ("Tester" wrote everything).
	h.press("/", "T", "e", "s", "t", "e", "r", "enter")
	if len(h.m().commits) != 5 {
		t.Fatalf("author search should match all commits, got %d", len(h.m().commits))
	}
	h.press("/", "n", "o", "b", "o", "d", "y", "enter")
	if len(h.m().commits) != 0 {
		t.Fatal("unmatched search should yield no commits")
	}
	h.press("esc")

	// ↑ at the top of a pane opens the search bar; ↓ leaves it again.
	h.press("2", "g", "up")
	if !h.m().searching {
		t.Fatal("↑ on the first row should focus the search bar")
	}
	h.press("down")
	if h.m().searching {
		t.Fatal("↓ should leave the search bar")
	}
	// Filter bar: → from the search box focuses the Branch chip, Enter opens
	// the picker, typing narrows it, Enter applies the branch filter.
	h.press("/", "right")
	m = h.m()
	if m.searching || !m.barBranch {
		t.Fatalf("→ at the end of the search text should focus the Branch chip (searching=%v chip=%v)", m.searching, m.barBranch)
	}
	if !strings.Contains(ansi.Strip(h.model.View()), "Branch: All ▾") {
		t.Fatal("branch chip should render with a dropdown marker")
	}
	h.press("enter")
	m = h.m()
	if m.menu == nil || !m.menu.filterable || len(m.menu.items) < 4 {
		t.Fatalf("branch picker should open with branches: %+v", m.menu)
	}
	h.press("s", "i", "d", "e")
	m = h.m()
	if m.menu.filter != "side" || m.menu.items[m.menu.cur].label != "feat/side" {
		t.Fatalf("typing should filter the picker: filter=%q cur=%q", m.menu.filter, m.menu.items[m.menu.cur].label)
	}
	h.press("enter")
	m = h.m()
	if m.logOpts.Ref != "feat/side" || m.barBranch || len(m.commits) != 3 {
		t.Fatalf("picking a branch should filter the log: ref=%q commits=%d", m.logOpts.Ref, len(m.commits))
	}
	// Esc on the focused chip clears the branch filter; ← goes back to search.
	h.press("/", "right", "esc")
	m = h.m()
	if m.logOpts.Ref != "" || !m.logOpts.All || m.barBranch {
		t.Fatalf("esc on the chip should clear the branch filter: %+v", m.logOpts)
	}
	h.press("/", "right", "left")
	if !h.m().searching {
		t.Fatal("← from the chip should return to the search box")
	}
	h.press("esc")
	if !strings.Contains(ansi.Strip(h.model.View()), "q exit") {
		t.Fatal("status bar should show q exit")
	}

	// ↓ past the last file hands over to the Details pane, ↑ comes back.
	h.press("3", "G", "down")
	if !h.m().detailsFocus {
		t.Fatal("↓ past the last file should focus Details")
	}
	h.press("down", "up", "up")
	if h.m().detailsFocus {
		t.Fatal("↑ at the top of Details should return to the files")
	}

	// Open the diff of the first file of the merge commit.
	h.press("j", "3", "enter")
	m = h.m()
	if m.diff == nil || m.diff.loading || len(m.diff.lines) == 0 {
		t.Fatalf("diff should be loaded: %+v", m.diff)
	}
	h.press("j", "ctrl+d", "n", "p", "esc")
	if h.m().diff != nil {
		t.Fatal("diff should be closed")
	}

	// Branch panel: fold/unfold, actions menu, show in log.
	h.press("1", "j", "j", " ")
	m = h.m()
	if !m.bexp["local/feat"] {
		t.Fatal("folder should be expanded")
	}
	h.press("j", "enter")
	if h.m().menu == nil {
		t.Fatal("branch menu should open")
	}
	h.press("s") // show in log
	m = h.m()
	if m.logOpts.Ref != "feat/side" || len(m.commits) != 3 {
		t.Fatalf("show-in-log failed: %q %d", m.logOpts.Ref, len(m.commits))
	}
	h.press("esc")

	// ← / → fold trees first, then move between panes (wrapping).
	h.press("2", "right")
	if h.m().focus != PanelChanges {
		t.Fatal("→ from the log should focus Changes")
	}
	h.press("right")
	if h.m().focus != PanelBranches {
		t.Fatal("→ from the rightmost pane should wrap to Branches")
	}
	h.press("g", "left")
	if h.m().focus != PanelChanges {
		t.Fatal("← from HEAD row should wrap to Changes")
	}
	h.press("1", "g", "j", "left") // Local section: collapse first…
	if m := h.m(); m.focus != PanelBranches || m.bexp["local"] {
		t.Fatalf("← on an expanded section should collapse it (focus=%v exp=%v)", m.focus, m.bexp["local"])
	}
	h.press("left") // …then leave the pane
	if h.m().focus != PanelChanges {
		t.Fatal("← on a collapsed section should move to the previous pane")
	}
	h.press("1", "right", "right") // re-expand Local, then → on an expanded section moves on
	if m := h.m(); !m.bexp["local"] || m.focus != PanelLog {
		t.Fatalf("→ should expand then move to the log (exp=%v focus=%v)", m.bexp["local"], m.focus)
	}
	if !strings.Contains(ansi.Strip(h.model.View()), "c commit") {
		t.Fatal("status bar should advertise c commit")
	}

	// Console + help overlays.
	h.press("`")
	if !h.m().console {
		t.Fatal("console should be visible")
	}
	h.press("`", "?")
	if !h.m().help {
		t.Fatal("help should be visible")
	}
	h.press("?")

	// Mouse: right-click a log row opens the menu; wheel scrolls.
	r := h.m().rects[PanelLog]
	h.mouse(r.x+10, r.y+2, tea.MouseButtonRight)
	m = h.m()
	if m.menu == nil || m.focus != PanelLog || m.lcur != 2 {
		t.Fatalf("right click: menu=%v focus=%v lcur=%d", m.menu != nil, m.focus, m.lcur)
	}
	h.press("esc")
	h.mouse(r.x+10, r.y+1, tea.MouseButtonWheelDown)
	if h.m().lcur == 2 {
		t.Fatal("wheel should move cursor")
	}

	// Commit workspace: main.go (tracked, checked by default) and new.txt
	// (unversioned, unchecked). Commit only new.txt, then main.go.
	h.press("g", "3")
	m = h.m()
	if !m.selected["main.go"] || m.selected["new.txt"] {
		t.Fatalf("default selection wrong: %+v", m.selected)
	}
	if len(m.fnodes) != 4 || !m.fnodes[0].isGroup || !m.fnodes[2].isGroup {
		t.Fatalf("expected Changes/Unversioned groups, got %+v", m.fnodes)
	}
	h.press("C")
	m = h.m()
	if !m.commitOpen || m.diff == nil || m.diff.path != "main.go" {
		t.Fatalf("commit workspace should open with main.go diff: open=%v diff=%+v", m.commitOpen, m.diff)
	}
	h.check("commit workspace")
	h.press(" ")           // uncheck main.go
	h.press("j", "j", " ") // check new.txt
	m = h.m()
	if m.selected["main.go"] || !m.selected["new.txt"] {
		t.Fatalf("selection toggles failed: %+v", m.selected)
	}
	h.press("ctrl+s") // no message yet → error toast, focus jumps to message
	m = h.m()
	if len(m.commits) != 5 || m.commit.focus != cfMessage {
		t.Fatalf("commit without message must be refused (commits=%d focus=%v)", len(m.commits), m.commit.focus)
	}
	h.press("t", "e", "s", "t", ":", " ", "커", "밋", "ctrl+s")
	m = h.m()
	if len(m.commits) != 6 {
		var hist []string
		for _, e := range m.repo.History() {
			if e.Err != nil {
				hist = append(hist, strings.Join(e.Args, " ")+" => "+e.Err.Error())
			}
		}
		t.Fatalf("expected 6 commits after commit, got %d; toast=%+v errors=%v", len(m.commits), m.toast, hist)
	}
	if m.commitOpen {
		t.Fatal("workspace should close after committing")
	}
	if len(m.status) != 1 || m.status[0].Path != "main.go" {
		t.Fatalf("main.go should remain modified (only new.txt committed), status=%+v", m.status)
	}
	if m.toast == nil || m.toast.kind != 1 {
		t.Fatalf("expected success toast, got %+v", m.toast)
	}
	// Second round: re-check main.go (unchecked state persists between opens),
	// recall the previous message with ↑, commit.
	h.press("C", " ", "enter", "up")
	m = h.m()
	if got := m.commit.msg.Value(); got != "test: 커밋" {
		t.Fatalf("history recall failed: %q", got)
	}
	h.press("ctrl+s")
	m = h.m()
	if len(m.commits) != 7 || len(m.status) != 0 {
		t.Fatalf("second commit failed: commits=%d status=%d", len(m.commits), len(m.status))
	}

	// Push dialog (no remote in this repo) and pull chooser.
	h.press("P")
	m = h.m()
	if m.push == nil || !strings.Contains(ansi.Strip(h.model.View()), "No remote configured") {
		t.Fatal("push dialog should explain the missing remote")
	}
	h.press("esc")
	if h.m().push != nil {
		t.Fatal("push dialog should close")
	}
	h.press("p")
	if h.m().menu == nil {
		t.Fatal("pull chooser should open")
	}
	h.press("esc")
	write(t, dir, "again.txt", "x\n")

	// New branch via commit menu prompt, then delete it from the branch panel.
	h.press("2", "j", "enter", "b")
	if h.m().dialog == nil {
		t.Fatal("branch prompt should open")
	}
	h.press("t", "m", "p", "enter")
	m = h.m()
	if m.info.Head != "tmp" {
		t.Fatalf("expected checkout of new branch tmp, got %q", m.info.Head)
	}
	if mm := h.m(); (&mm).selectedCommit() == nil {
		t.Fatal("cursor should stay on a commit after reload")
	}

	// Narrow terminal must still satisfy the invariant.
	h.model = drain(h.model, tea.WindowSizeMsg{Width: 90, Height: 20}, 0)
	h.w, h.h = 90, 20
	h.check("narrow")
	h.press("j", "k", "tab", "tab", "j")
}

func TestNotARepo(t *testing.T) {
	m := New(t.TempDir())
	if m.fatal == nil {
		t.Fatal("expected fatal error outside a repo")
	}
	m.width, m.height = 80, 20
	if !strings.Contains(ansi.Strip(m.View()), "could not open") {
		t.Fatal("expected error screen")
	}
}
