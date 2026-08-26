package ui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestMenuScrollAndFilter(t *testing.T) {
	var items []menuItem
	for i := 0; i < 12; i++ {
		items = append(items, menuItem{label: fmt.Sprintf("branch-%02d", i)})
	}
	m := Model{width: 80, height: 20}
	mn := &menu{title: "Branch", items: items, filterable: true}
	m.openMenu(mn, 0, 10) // only ~6 rows fit below y=10
	if mn.maxRows == 0 || mn.maxRows >= len(items) {
		t.Fatalf("menu should be capped: maxRows=%d", mn.maxRows)
	}
	for i := 0; i < 10; i++ {
		mn.move(1)
	}
	mn.render()
	if mn.cur != 10 || mn.scroll != 10-mn.maxRows+1 {
		t.Fatalf("cursor should stay visible: cur=%d scroll=%d rows=%d", mn.cur, mn.scroll, mn.maxRows)
	}
	// itemAt maps screen rows through the scroll offset.
	if got := mn.itemAt(mn.x+2, mn.y+1+2); got != mn.scroll {
		t.Fatalf("itemAt first visible row = %d, want %d", got, mn.scroll)
	}
	// Typing filters; an impossible query shows "no matches".
	mn.filter = "branch-1"
	mn.refilter()
	if len(mn.items) != 2 || mn.cur != 0 {
		t.Fatalf("filter should keep branch-10/11: %d items cur=%d", len(mn.items), mn.cur)
	}
	mn.filter = "zzz"
	mn.refilter()
	if !strings.Contains(ansi.Strip(mn.render()), "no matches") {
		t.Fatal("empty filter result should say no matches")
	}
}
