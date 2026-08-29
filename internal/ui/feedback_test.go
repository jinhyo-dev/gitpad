package ui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	"github.com/jinhyo-dev/gitpad/internal/git"
)

// Progress box and notification banner are drawn as centered overlays.
func TestProgressAndToastOverlays(t *testing.T) {
	dir := makeRepo(t)
	h := newHarness(t, dir, 120, 30)
	m := h.m()
	m.actions, m.actionLabel = 1, "Fetch"
	m.toast = &toast{id: 1, text: "Push main", kind: 1}
	h.model = m
	h.check("overlays")
	view := ansi.Strip(h.model.View())
	lines := strings.Split(view, "\n")
	if !strings.Contains(view, "Fetch…") {
		t.Fatal("progress box should show the running action")
	}
	if !strings.Contains(lines[rowPanels], "✓  Push main") {
		t.Fatalf("toast banner should sit top-center on row %d: %q", rowPanels, lines[rowPanels])
	}
	if out := os.Getenv("GITPAD_DUMP"); out != "" {
		os.WriteFile(filepath.Join(out, "11-feedback.txt"), []byte(view), 0o644)
	}
}

func TestDiffChangeBlockNavigation(t *testing.T) {
	text := "diff --git a/f b/f\n--- a/f\n+++ b/f\n@@ -1,8 +1,8 @@\n a\n-b\n+B\n c\n d\n e\n f\n-g\n+G\n+H\n"
	lines, _ := parseDiff(text)
	blocks := changeBlocks(lines)
	if len(blocks) != 2 {
		t.Fatalf("expected 2 change blocks, got %v", blocks)
	}
	m := &Model{}
	m.diffRect = rect{h: 6}
	m.diff = &diffState{lines: lines, blocks: blocks}
	m.diffJump(1)
	if m.diff.cur != 1 || m.diff.scroll != clamp(blocks[1][0]-2, 0, m.diffMaxScroll()) {
		t.Fatalf("jump forward: cur=%d scroll=%d", m.diff.cur, m.diff.scroll)
	}
	m.diffJump(1) // stays on the last block
	if m.diff.cur != 1 {
		t.Fatal("must not run past the last block")
	}
	m.diffJump(-1)
	if m.diff.cur != 0 || m.diff.scroll != clamp(blocks[0][0]-2, 0, m.diffMaxScroll()) {
		t.Fatalf("jump back: cur=%d scroll=%d", m.diff.cur, m.diff.scroll)
	}
	if !m.diff.inCurrentBlock(blocks[0][0]) || m.diff.inCurrentBlock(blocks[1][0]) {
		t.Fatal("current block membership wrong")
	}
}

func TestSemverHelpers(t *testing.T) {
	for in, want := range map[string]string{"v1.2.3": "v1.2.3", "0.9.10": "0.9.10"} {
		v, ok := parseSemver(in)
		if !ok || v.String() != want {
			t.Errorf("%s: %v %v", in, v, ok)
		}
	}
	for _, bad := range []string{"v1.2", "1.2.3.4", "va.b.c", "release-1"} {
		if _, ok := parseSemver(bad); ok {
			t.Errorf("%s should not parse", bad)
		}
	}
	latest, ok := latestSemver([]git.Branch{{Name: "v0.2.0"}, {Name: "v0.10.1"}, {Name: "v0.9.9"}, {Name: "nightly"}})
	if !ok || latest.String() != "v0.10.1" {
		t.Fatalf("latest = %v %v", latest, ok)
	}
}

func TestVersionTagFlow(t *testing.T) {
	dir := makeRepo(t) // has v0.1.0
	h := newHarness(t, dir, 120, 30)
	h.press("v")
	m := h.m()
	if m.menu == nil || !strings.Contains(m.menu.title, "v0.1.0") || m.menu.cur != 1 {
		t.Fatalf("version menu should open on the minor bump: %+v", m.menu)
	}
	if !strings.HasPrefix(m.menu.items[1].label, "v0.2.0") {
		t.Fatalf("minor suggestion wrong: %q", m.menu.items[1].label)
	}
	h.press("enter") // choose v0.2.0
	if h.m().dialog == nil || !h.m().dialog.allowEmpty {
		t.Fatal("annotation prompt should open and accept an empty message")
	}
	h.press("R", "e", "l", "e", "a", "s", "e", " ", "0", ".", "2", "enter")
	m = h.m()
	found := false
	for _, tg := range m.refs.Tags {
		if tg.Name == "v0.2.0" {
			found = true
		}
	}
	if !found {
		t.Fatalf("v0.2.0 should exist, tags=%+v", m.refs.Tags)
	}
	out, _ := m.repo.Run("tag", "-n1", "v0.2.0")
	if !strings.Contains(out, "Release 0.2") {
		t.Fatalf("tag should be annotated with the message: %q", out)
	}
	if m.dialog != nil { // no remote in this repo → no push offer
		t.Fatalf("no push offer expected without a remote, got %+v", m.dialog)
	}
}

// Hunk staging: uncheck one hunk of a two-hunk file and commit → HEAD gets
// only the other hunk, the working tree keeps both.
func TestHunkStaging(t *testing.T) {
	dir := makeRepo(t)
	var b strings.Builder
	for i := 1; i <= 40; i++ {
		fmt.Fprintf(&b, "line %d\n", i)
	}
	base := b.String()
	write(t, dir, "big.txt", base)
	run(t, dir, "add", "big.txt")
	run(t, dir, "commit", "-q", "-m", "add big.txt")
	ls := strings.Split(strings.TrimRight(base, "\n"), "\n")
	ls[1], ls[37] = "TOP", "BOTTOM"
	write(t, dir, "big.txt", strings.Join(ls, "\n")+"\n")

	h := newHarness(t, dir, 150, 40)
	m := h.m()
	if !m.selected["big.txt"] {
		t.Fatal("tracked change should start checked")
	}
	// Open the workspace, move the file cursor onto big.txt, focus the diff.
	h.press("C")
	for i, n := range h.m().fnodes {
		if !n.isDir && n.file.Path == "big.txt" {
			mm := h.m()
			mm.fcur = i
			h.model = mm
		}
	}
	h.model = drain(h.model, keyMsg("3"), 0) // focus diff (loads big.txt's diff)
	m = h.m()
	if m.diff == nil || m.diff.path != "big.txt" || len(m.diff.hunks) != 2 {
		t.Fatalf("expected a two-hunk diff for big.txt, got %+v", m.diff)
	}
	h.press(" ") // uncheck hunk 0 (cursor advances to hunk 1)
	m = h.m()
	if set := m.hunkSel["big.txt"]; set == nil || set[0] || !set[1] {
		t.Fatalf("hunk 0 should be unchecked, hunk 1 checked: %+v", set)
	}
	if !strings.Contains(ansi.Strip(h.model.View()), "[~]") {
		t.Fatal("partially checked file should show [~]")
	}
	// Uncheck the other dirty files so only big.txt is committed.
	mm := h.m()
	for _, f := range mm.status {
		if f.Path != "big.txt" {
			mm.setFileSelected(f.Path, false)
		}
	}
	h.model = mm
	h.press("2", "h", "u", "n", "k", "ctrl+s")
	m = h.m()
	head, _ := m.repo.Run("show", "HEAD:big.txt")
	if strings.Contains(head, "TOP") || !strings.Contains(head, "BOTTOM") {
		t.Fatalf("HEAD should contain only the second hunk:\n%s", head)
	}
	rest, _ := m.repo.Run("diff", "HEAD", "--", "big.txt")
	if !strings.Contains(rest, "TOP") || strings.Contains(rest, "BOTTOM") {
		t.Fatalf("remaining diff should be the first hunk only:\n%s", rest)
	}
	if _, partial := m.hunkSel["big.txt"]; partial {
		t.Fatal("hunk selection should reset after the commit")
	}
}

func TestCommandPalette(t *testing.T) {
	dir := makeRepo(t)
	h := newHarness(t, dir, 150, 40)
	h.press("j", "j") // a non-HEAD commit → cherry-pick is available
	// Put the branch cursor on the "main" leaf so branch actions join too.
	mm := h.m()
	for i, n := range mm.bnodes {
		if n.kind == bLeaf && n.branch != nil && n.branch.Name == "main" {
			mm.bcur = i
		}
	}
	h.model = mm
	h.model = drain(h.model, tea.KeyMsg{Type: tea.KeyCtrlK}, 0)
	m := h.m()
	if m.menu == nil || !m.menu.filterable || m.menu.title != "Command palette" {
		t.Fatalf("ctrl+k should open the palette: %+v", m.menu)
	}
	var labels []string
	for _, it := range m.menu.items {
		labels = append(labels, it.label)
	}
	joined := strings.Join(labels, "\n")
	for _, want := range []string{"Push…", "Keyboard help", "› Cherry-pick", "› Reset main to here › Soft", "Branch main › Show in log", "File "} {
		if !strings.Contains(joined, want) {
			t.Errorf("palette missing %q in:\n%s", want, joined)
		}
	}
	// Type to filter, Enter runs the action.
	h.press("h", "e", "l", "p")
	m = h.m()
	if len(m.menu.items) != 1 || m.menu.items[0].label != "Keyboard help" {
		t.Fatalf("filter should narrow to the help entry: %+v", m.menu.items)
	}
	h.press("enter")
	if m := h.m(); m.menu != nil || !m.help {
		t.Fatal("running the entry should open the help overlay")
	}
}

// Interactive rebase workspace: drop one commit, reword another, run.
func TestInteractiveRebaseWorkspace(t *testing.T) {
	dir := t.TempDir()
	run(t, dir, "init", "-q", "-b", "main")
	for _, name := range []string{"one", "two", "three", "four"} {
		write(t, dir, name+".txt", name+"\n")
		run(t, dir, "add", ".")
		run(t, dir, "commit", "-q", "-m", "commit "+name)
	}
	h := newHarness(t, dir, 150, 40)
	h.press("j", "j") // "two" (rows: four, three, two, one)
	h.press("i")
	m := h.m()
	if !m.rebaseOpen || m.rebase == nil || len(m.rebase.steps) != 3 {
		t.Fatalf("rebase workspace should open with 3 commits: open=%v %+v", m.rebaseOpen, m.rebase)
	}
	h.check("rebase workspace")
	// Cursor starts on the newest ("four"): drop "three", reword "two".
	h.press("j", "d", "j", "r")
	if h.m().dialog == nil {
		t.Fatal("reword should prompt for a message")
	}
	// Prompt is prefilled with the subject; replace it.
	dl := h.m()
	dl.dialog.input.SetValue("")
	h.model = dl
	h.press("T", "W", "O", "enter")
	m = h.m()
	if m.rebase.steps[1].Action != "drop" || m.rebase.steps[2].Action != "reword" || m.rebase.steps[2].Message != "TWO" {
		t.Fatalf("plan not as expected: %+v", m.rebase.steps)
	}
	h.press("ctrl+s")
	if h.m().dialog == nil {
		t.Fatal("start should ask for confirmation")
	}
	h.press("enter")
	m = h.m()
	if m.rebaseOpen {
		t.Fatal("workspace should close after the rebase")
	}
	var subjects []string
	for _, c := range m.commits {
		subjects = append(subjects, c.Subject)
	}
	if got := strings.Join(subjects, "|"); got != "commit four|TWO|commit one" {
		t.Fatalf("history after rebase: %s (toast=%+v)", got, m.toast)
	}
}

// Undo walks back commits, resets and branch deletions.
func TestUndo(t *testing.T) {
	dir := makeRepo(t)
	h := newHarness(t, dir, 150, 40)
	h.press("u")
	if h.m().dialog != nil {
		t.Fatal("nothing to undo yet — no dialog expected")
	}
	// 1) Commit everything, then undo it: the commit disappears, changes stay.
	h.press("C", "a", "tab", "w", "i", "p", "ctrl+s")
	m := h.m()
	if len(m.commits) != 6 {
		t.Fatalf("expected 6 commits after committing, got %d", len(m.commits))
	}
	h.press("u")
	m = h.m()
	if m.dialog == nil || !strings.Contains(m.dialog.title, "Undo Commit") {
		t.Fatalf("undo should ask about the commit: %+v", m.dialog)
	}
	h.press("enter")
	m = h.m()
	if len(m.commits) != 5 || len(m.status) == 0 {
		t.Fatalf("commit should be undone with changes kept: commits=%d status=%d", len(m.commits), len(m.status))
	}
	// 2) Delete a branch from the Branches pane, then undo re-creates it.
	// Branches: HEAD, Local, ▸ feat (collapsed), … → unfold feat, select side.
	h.press("1", "g", "j", "j", "right", "j")
	if mm := h.m(); (&mm).selectedBranchNode() == nil || (&mm).selectedBranchNode().branch == nil || (&mm).selectedBranchNode().branch.Name != "feat/side" {
		t.Fatalf("expected feat/side selected, got %+v", h.m().bnodes[h.m().bcur])
	}
	h.press("d")
	if d := h.m().dialog; d == nil {
		mm := h.m()
		t.Fatalf("delete should ask first (focus=%v bcur=%d node=%+v toast=%+v)", mm.focus, mm.bcur, mm.bnodes[mm.bcur], mm.toast)
	}
	h.press("enter")
	m = h.m()
	for _, b := range m.refs.Locals {
		if b.Name == "feat/side" {
			var errs []string
			for _, e := range m.repo.History() {
				if e.Err != nil {
					errs = append(errs, strings.Join(e.Args, " ")+": "+e.Err.Error())
				}
			}
			t.Fatalf("branch should be deleted (toast=%+v dialog=%v errors=%v)", m.toast, m.dialog != nil, errs)
		}
	}
	h.press("u", "enter")
	m = h.m()
	found := false
	for _, b := range m.refs.Locals {
		if b.Name == "feat/side" {
			found = true
		}
	}
	if !found {
		t.Fatalf("undo should re-create the branch (toast=%+v)", m.toast)
	}
	// 3) Hard reset to an older commit, then undo brings HEAD back.
	head := m.info.HeadHash
	h.press("2", "j", "j", "enter", "r") // commit menu → Reset ▸
	h.press("j", "j", "enter")           // Hard
	h.press("enter")                     // confirm
	m = h.m()
	if m.info.HeadHash == head {
		t.Fatal("reset should have moved HEAD")
	}
	h.press("u", "enter")
	if m := h.m(); m.info.HeadHash != head {
		t.Fatalf("undo should restore HEAD %s, got %s (toast=%+v)", head[:8], m.info.HeadHash[:8], m.toast)
	}
}

// Conflict resolution: a conflicting merge is resolved block by block and
// continued from the workspace.
func TestConflictWorkspace(t *testing.T) {
	dir := t.TempDir()
	run(t, dir, "init", "-q", "-b", "main")
	write(t, dir, "f.txt", "one\ntwo\nthree\n")
	run(t, dir, "add", ".")
	run(t, dir, "commit", "-q", "-m", "base")
	run(t, dir, "checkout", "-q", "-b", "feature")
	write(t, dir, "f.txt", "one\nTHEIRS\nthree\n")
	run(t, dir, "commit", "-q", "-am", "theirs")
	run(t, dir, "checkout", "-q", "main")
	write(t, dir, "f.txt", "one\nOURS\nthree\n")
	run(t, dir, "commit", "-q", "-am", "ours")
	cmd := exec.Command("git", "merge", "feature")
	cmd.Dir = dir
	_ = cmd.Run() // conflicts on purpose

	h := newHarness(t, dir, 150, 40)
	m := h.m()
	if m.info.State != "merging" || !(&m).hasConflicts() {
		t.Fatalf("expected a merging repo with conflicts: state=%q status=%+v", m.info.State, m.status)
	}
	if m.toast == nil || !strings.Contains(m.toast.text, "conflicted file") {
		t.Fatalf("conflicts should be announced, toast=%+v", m.toast)
	}
	h.press("x")
	m = h.m()
	if !m.conflictOpen || m.conflict == nil || len(m.conflict.blocks) != 1 {
		t.Fatalf("workspace should open with one conflict: %+v", m.conflict)
	}
	h.check("conflict workspace")
	view := ansi.Strip(h.model.View())
	if !strings.Contains(view, "◀ ours") || !strings.Contains(view, "▶ theirs") {
		t.Fatal("conflict headers should be rendered")
	}
	h.press("ctrl+s") // unresolved → refused
	if h.m().conflict == nil || h.m().toast == nil || !strings.Contains(h.m().toast.text, "unresolved") {
		t.Fatalf("saving with unresolved blocks must be refused: %+v", h.m().toast)
	}
	h.press("b", "ctrl+s") // keep both, save → all resolved → continue offer
	m = h.m()
	if m.dialog == nil || !strings.Contains(m.dialog.title, "All conflicts resolved") {
		t.Fatalf("expected the continue offer, dialog=%+v conflictOpen=%v", m.dialog, m.conflictOpen)
	}
	h.press("enter")
	m = h.m()
	if m.info.State != "" {
		t.Fatalf("merge should be continued, state=%q toast=%+v", m.info.State, m.toast)
	}
	content, _ := m.repo.Run("show", "HEAD:f.txt")
	if content != "one\nOURS\nTHEIRS\nthree\n" {
		t.Fatalf("merged file content: %q", content)
	}
}
