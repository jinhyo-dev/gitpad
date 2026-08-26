package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/jinhyo/gitpad/internal/ui/theme"
)

type menuItem struct {
	label    string
	key      string
	danger   bool
	sep      bool
	disabled bool
	run      func(m *Model) tea.Cmd
	sub      func(m *Model) *menu
}

type menu struct {
	title  string
	items  []menuItem
	cur    int
	x, y   int
	w, h   int
	parent *menu
}

func sep() menuItem { return menuItem{sep: true} }

func (mn *menu) move(d int) {
	if len(mn.items) == 0 {
		return
	}
	for i := 0; i < len(mn.items); i++ {
		mn.cur = (mn.cur + d + len(mn.items)) % len(mn.items)
		if !mn.items[mn.cur].sep && !mn.items[mn.cur].disabled {
			return
		}
	}
}

func (mn *menu) first() {
	mn.cur = -1
	mn.move(1)
}

// layout measures the box; must be called before render/positioning.
func (mn *menu) layout() {
	lw, kw := width(mn.title), 0
	for _, it := range mn.items {
		if it.sep {
			continue
		}
		l := it.label
		if it.sub != nil {
			l += " ▸"
		}
		lw = maxInt(lw, width(l))
		kw = maxInt(kw, width(it.key))
	}
	mn.w = lw + kw + 3 + 4 // gap + border/padding
	mn.h = len(mn.items) + 2
	if mn.title != "" {
		mn.h += 2
	}
}

func (mn *menu) render() string {
	inner := mn.w - 4
	var lines []string
	if mn.title != "" {
		lines = append(lines, pad(theme.MenuTitle.Render(trunc(mn.title, inner)), inner))
		lines = append(lines, theme.MenuSep.Render(strings.Repeat("─", inner)))
	}
	for i, it := range mn.items {
		if it.sep {
			lines = append(lines, theme.MenuSep.Render(strings.Repeat("─", inner)))
			continue
		}
		label := it.label
		if it.sub != nil {
			label += " ▸"
		}
		st := theme.MenuItem
		switch {
		case it.disabled:
			st = theme.MenuDisabled
		case it.danger:
			st = theme.MenuDanger
		}
		key := theme.MenuKey.Render(it.key)
		gap := inner - width(label) - width(it.key)
		if gap < 1 {
			gap = 1
		}
		line := st.Render(label) + strings.Repeat(" ", gap) + key
		if i == mn.cur {
			line = highlight(lipgloss.NewStyle().Bold(true).Render(st.Render(label))+strings.Repeat(" ", gap)+key, theme.Selection)
		}
		lines = append(lines, pad(line, inner))
	}
	return theme.MenuBox.Render(strings.Join(lines, "\n"))
}

// itemAt maps a screen coordinate to an item index (-1 when outside).
func (mn *menu) itemAt(x, y int) int {
	if x < mn.x || x >= mn.x+mn.w || y < mn.y || y >= mn.y+mn.h {
		return -1
	}
	row := y - mn.y - 1
	if mn.title != "" {
		row -= 2
	}
	if row < 0 || row >= len(mn.items) {
		return -1
	}
	if mn.items[row].sep || mn.items[row].disabled {
		return -1
	}
	return row
}

func (mn *menu) inside(x, y int) bool {
	return x >= mn.x && x < mn.x+mn.w && y >= mn.y && y < mn.y+mn.h
}

// openMenu shows a menu anchored at (x, y), clamped into the screen.
func (m *Model) openMenu(mn *menu, x, y int) {
	if mn == nil || len(mn.items) == 0 {
		return
	}
	mn.layout()
	mn.first()
	if x+mn.w > m.width {
		x = m.width - mn.w
	}
	if y+mn.h > m.height {
		y = m.height - mn.h
	}
	mn.x, mn.y = maxInt(0, x), maxInt(0, y)
	m.menu = mn
}

func (m *Model) menuAnchorFor(p Panel, row int) (int, int) {
	if m.menuAnchor != nil {
		return m.menuAnchor[0], m.menuAnchor[1]
	}
	r := m.rects[p]
	return r.x + 2, r.y + row - m.scrollOf(p) + 1
}

func (m *Model) scrollOf(p Panel) int {
	switch p {
	case PanelBranches:
		return m.bscroll
	case PanelLog:
		return m.lscroll
	default:
		return m.fscroll
	}
}

func (m *Model) closeMenu() { m.menu = nil }

// activateMenuItem runs the selected item.
func (m *Model) activateMenuItem() tea.Cmd {
	mn := m.menu
	if mn == nil || mn.cur < 0 || mn.cur >= len(mn.items) {
		return nil
	}
	it := mn.items[mn.cur]
	if it.disabled || it.sep {
		return nil
	}
	if it.sub != nil {
		sub := it.sub(m)
		if sub == nil {
			return nil
		}
		sub.parent = mn
		m.openMenu(sub, mn.x+4, mn.y+mn.cur+1)
		return nil
	}
	m.menu = nil
	if it.run != nil {
		return it.run(m)
	}
	return nil
}
