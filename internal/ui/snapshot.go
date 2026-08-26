package ui

import (
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

// Snapshot renders a single frame synchronously (no TTY needed). Commands
// returned by Update are executed inline so the frame reflects loaded data.
func Snapshot(path string, w, h int) string {
	m := New(path)
	if m.fatal != nil {
		m.width, m.height = w, h
		return m.View()
	}
	var model tea.Model = m
	model = drain(model, tea.WindowSizeMsg{Width: w, Height: h}, 0)
	model = drain(model, m.Init()(), 0)
	return model.View()
}

func drain(model tea.Model, msg tea.Msg, depth int) tea.Model {
	if msg == nil || depth > 8 {
		return model
	}
	switch b := msg.(type) {
	case tea.BatchMsg:
		for _, c := range b {
			if c != nil {
				model = drain(model, c(), depth+1)
			}
		}
		return model
	case watchTickMsg, watchResultMsg, toastClearMsg:
		return model
	}
	var cmd tea.Cmd
	model, cmd = model.Update(msg)
	if cmd == nil {
		return model
	}
	next := cmd()
	switch next.(type) {
	case nil:
		return model
	}
	// Skip timers (ticks) — they'd block the snapshot.
	if _, ok := next.(tea.BatchMsg); !ok {
		if isTick(next) {
			return model
		}
	}
	return drain(model, next, depth+1)
}

func isTick(msg tea.Msg) bool {
	switch msg.(type) {
	case watchTickMsg, toastClearMsg, spinner.TickMsg:
		return true
	}
	return false
}
