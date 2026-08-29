package ui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/jinhyo-dev/gitpad/internal/git"
)

type logOptionsType = git.LogOptions

// The command palette (ctrl+k) lists every action that makes sense right
// now — global commands plus the context menus of whatever is selected —
// as one type-to-filter list, so nothing requires memorising a key.

func (m *Model) commandPalette() tea.Cmd {
	var items []menuItem
	add := func(label, key string, run func(m *Model) tea.Cmd) {
		items = append(items, menuItem{label: label, key: key, run: run})
	}

	// Global commands.
	if len(m.status) > 0 {
		add("Commit…", "c", func(m *Model) tea.Cmd { return m.openCommit() })
	}
	add("Push…", "P", func(m *Model) tea.Cmd { return m.openPush() })
	add("Pull…", "p", func(m *Model) tea.Cmd { return m.pullChooser() })
	add("Fetch (all, prune)", "f", func(m *Model) tea.Cmd { return m.action("Fetch", m.repo.Fetch) })
	add("New version tag…", "v", func(m *Model) tea.Cmd { return m.versionTagMenu() })
	add("Search message, author or hash", "/", func(m *Model) tea.Cmd { return m.startSearch() })
	add("Choose branch filter…", "", func(m *Model) tea.Cmd { m.barBranch = true; m.openBranchPicker(); return nil })
	if m.logOpts.All {
		add("Show current branch only", "A", func(m *Model) tea.Cmd { m.logOpts.All = false; m.logOpts.Ref = ""; return m.applyFilter() })
	} else {
		add("Show all branches", "A", func(m *Model) tea.Cmd { return m.showRefInLog("") })
	}
	if m.hasFilter() {
		add("Clear filters", "esc", func(m *Model) tea.Cmd { m.logOpts = defaultLogOptions(); return m.applyFilter() })
	}
	add("Refresh", "r", func(m *Model) tea.Cmd { return m.reload() })
	add("Console (git commands)", "`", func(m *Model) tea.Cmd { m.console = !m.console; m.focus = PanelLog; return nil })
	if base := remoteWebURL(m.repo); base != "" {
		add("Open repository in browser", "", func(m *Model) tea.Cmd { return openBrowser(base) })
	}
	add("Keyboard help", "?", func(m *Model) tea.Cmd { m.help = true; return nil })
	add("Quit", "q", func(m *Model) tea.Cmd { m.confirmQuit(); return nil })

	// Context: whatever is selected in each pane.
	flatten := func(prefix string, mn *menu) {
		if mn == nil {
			return
		}
		for _, it := range mn.items {
			if it.sep || it.disabled {
				continue
			}
			label := prefix + " › " + it.label
			if it.sub != nil {
				sub := it.sub
				if s := sub(m); s != nil {
					for _, si := range s.items {
						if si.sep || si.disabled {
							continue
						}
						items = append(items, menuItem{label: label + " › " + si.label, danger: si.danger, run: si.run})
					}
				}
				continue
			}
			items = append(items, menuItem{label: label, danger: it.danger, run: it.run})
		}
	}
	if m.isLocalRow(m.lcur) {
		flatten("Changes", m.menuForLocal())
	} else if c := m.selectedCommit(); c != nil {
		flatten("Commit "+c.Short, m.menuForCommit(c))
	}
	if n := m.selectedBranchNode(); n != nil && n.branch != nil {
		flatten("Branch "+n.branch.Name, m.branchActions(n.branch))
	}
	if n := m.selectedFileNode(); n != nil && !n.isDir {
		flatten("File "+n.label, m.menuForFile(n))
	}

	mn := &menu{title: "Command palette", items: items, filterable: true}
	mn.layout()
	m.openMenu(mn, (m.width-mn.w)/2, 3)
	return nil
}

func defaultLogOptions() logOptionsType { return logOptionsType{All: true, Limit: 1000} }
