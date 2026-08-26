package ui

import (
	"os"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// TestRenderCost measures View() and a key press on a real repository.
// Enabled with GITPAD_BENCH_REPO=<path>.
func TestRenderCost(t *testing.T) {
	repo := os.Getenv("GITPAD_BENCH_REPO")
	if repo == "" {
		t.Skip("set GITPAD_BENCH_REPO")
	}
	h := newHarness(t, repo, 180, 50)
	start := time.Now()
	for i := 0; i < 50; i++ {
		_ = h.model.View()
	}
	t.Logf("View(): %v per frame", time.Since(start)/50)

	start = time.Now()
	for i := 0; i < 20; i++ {
		h.model, _ = h.model.Update(keyMsg("j"))
		_ = h.model.View()
	}
	t.Logf("Update(j)+View (no git): %v per step", time.Since(start)/20)

	start = time.Now()
	for i := 0; i < 10; i++ {
		h.model = drain(h.model, keyMsg("j"), 0)
	}
	t.Logf("Update(j)+git file load+View: %v per step", time.Since(start)/10)

	mm := h.model.(Model)
	start = time.Now()
	msg := mm.loadAll("")()
	t.Logf("loadAll: %v", time.Since(start))
	_ = msg
	_ = tea.KeyMsg{}
}
