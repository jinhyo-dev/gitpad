package ui

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
)

// TestDumpScreens writes plain-text renders of key screens for eyeballing.
// Enabled with GITPAD_DUMP=<dir>.
func TestDumpScreens(t *testing.T) {
	out := os.Getenv("GITPAD_DUMP")
	if out == "" {
		t.Skip("set GITPAD_DUMP to write screen dumps")
	}
	toastDuration, toastErrDuration, watchInterval = time.Millisecond, time.Millisecond, time.Millisecond
	dir := makeRepo(t)
	h := newHarness(t, dir, 150, 34)
	save := func(name string) {
		os.WriteFile(filepath.Join(out, name+".txt"), []byte(ansi.Strip(h.model.View())), 0o644)
	}
	h.press("j")
	save("1-log")
	h.press("enter")
	save("2-menu")
	h.press("r")
	save("3-submenu")
	h.press("esc", "esc", "enter", "b")
	save("4-dialog")
	h.press("esc", "3", "enter", "j")
	save("5-diff")
	h.press("esc", "?")
	save("6-help")
	h.press("?", "1", "j", "j", " ", "j", "enter")
	save("7-branch-menu")
	h.press("esc", "2", "g", "C", "j", "j", " ", "enter", "f", "e", "a", "t", ":", " ", "커", "밋", " ", "메", "시", "지")
	save("8-commit")
	h.press("esc", "esc", "P")
	save("9-push")
}
