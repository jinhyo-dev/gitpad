package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// The push dialog must not disturb the background outside its box.
func TestOverlayPreservesBackground(t *testing.T) {
	dir := makeRepo(t)
	h := newHarness(t, dir, 150, 34)
	plain := strings.Split(ansi.Strip(h.model.View()), "\n")
	h.press("P")
	m := h.m()
	if m.push == nil {
		t.Fatal("push dialog should be open")
	}
	x, y, w, bh := (&m).pushBox()
	over := strings.Split(ansi.Strip(h.model.View()), "\n")
	for row := y; row < y+bh; row++ {
		left := ansi.Truncate(plain[row], x, "")
		gotLeft := ansi.Truncate(over[row], x, "")
		if left != gotLeft {
			t.Errorf("row %d left differs:\n  want %q\n  got  %q", row, left, gotLeft)
		}
		right := ansi.TruncateLeft(plain[row], x+w, "")
		gotRight := ansi.TruncateLeft(over[row], x+w, "")
		if right != gotRight {
			t.Errorf("row %d right differs (box x=%d w=%d):\n  want %q\n  got  %q", row, x, w, right, gotRight)
		}
	}
}
