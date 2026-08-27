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
