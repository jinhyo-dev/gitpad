package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
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
