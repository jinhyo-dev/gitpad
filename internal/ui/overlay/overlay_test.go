package overlay

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestComposeKeepsWidthWithWideGlyphs(t *testing.T) {
	bg := strings.Repeat("가", 20) // 40 cells
	fg := "[MENU]"
	for x := 0; x < 40; x++ {
		out := Compose(bg, fg, x, 0)
		if w := ansi.StringWidth(out); w != 40 {
			t.Fatalf("x=%d width=%d: %q", x, w, out)
		}
		if x <= 34 && !strings.Contains(out, fg) {
			t.Fatalf("x=%d overlay missing", x)
		}
	}
}
