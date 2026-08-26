package ui

import (
	"context"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/jinhyo-dev/gitpad/internal/ci"
)

// fakeCI answers success / failure / pending for the first three hashes it
// is asked about and records how many requests were made.
type fakeCI struct{ calls int }

func (f *fakeCI) Name() string                { return "fake" }
func (f *fakeCI) ChecksURL(sha string) string { return "https://example.test/" + sha }
func (f *fakeCI) Fetch(_ context.Context, shas []string) (map[string]ci.Result, error) {
	f.calls++
	out := map[string]ci.Result{}
	states := []ci.State{ci.StateSuccess, ci.StateFailure, ci.StatePending}
	for i, s := range shas {
		st := ci.StateNone
		if i < len(states) {
			st = states[i]
		}
		out[s] = ci.Result{State: st, Checks: []ci.Check{{Name: "test (ubuntu)", State: st, Duration: "1m 02s"}}}
	}
	return out, nil
}

func TestCIColumn(t *testing.T) {
	dir := makeRepo(t)
	h := newHarness(t, dir, 150, 34)
	fake := &fakeCI{}
	h.model = drain(h.model, ciInitMsg{p: fake}, 0)
	h.check("after ci init")
	m := h.m()
	if m.ci == nil || fake.calls != 1 || len(m.ciResults) != 5 {
		t.Fatalf("ci not fetched: provider=%v calls=%d results=%d", m.ci != nil, fake.calls, len(m.ciResults))
	}
	view := ansi.Strip(h.model.View())
	for _, g := range []string{"✓", "✗", "◌"} {
		if !strings.Contains(view, g) {
			t.Errorf("view missing CI glyph %q", g)
		}
	}
	// Details lists the checks of the selected commit.
	h.press("j")
	if v := ansi.Strip(h.model.View()); !strings.Contains(v, "Checks") || !strings.Contains(v, "test (ubuntu)") {
		t.Error("details should list checks")
	}
	// Navigating must not trigger new requests (everything is cached).
	h.press("j", "j", "k")
	if fake.calls != 1 {
		t.Errorf("expected cached results, got %d calls", fake.calls)
	}
	// Commit menu offers the checks link.
	h.press("enter")
	found := false
	for _, it := range h.m().menu.items {
		if it.label == "Open checks in browser" {
			found = true
		}
	}
	if !found {
		t.Error("menu should offer 'Open checks in browser'")
	}
	h.press("esc")
	// A refresh tick drops pending results so they are re-fetched.
	h.model = drain(h.model, ciRefreshMsg{}, 0)
	if fake.calls != 2 {
		t.Errorf("pending commits should be refetched after the tick, calls=%d", fake.calls)
	}
}
