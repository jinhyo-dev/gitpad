package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

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
