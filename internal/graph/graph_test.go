package graph

import (
	"strings"
	"testing"
)

func render(rows []Row) string {
	var sb strings.Builder
	for _, r := range rows {
		for _, c := range r.Cells {
			sb.WriteRune(c.Rune)
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

func TestBuildShapes(t *testing.T) {
	// A merge of a side branch, a second branch head, and a root.
	//   m (merge of a + s2)
	//   a
	//   s2 -> s1 -> b   (side branch)
	//   h2 (other head) -> b
	//   b -> root
	nodes := []Node{
		{"m", []string{"a", "s2"}},
		{"h2", []string{"b"}},
		{"a", []string{"b"}},
		{"s2", []string{"s1"}},
		{"s1", []string{"b"}},
		{"b", []string{"root"}},
		{"root", nil},
	}
	rows := Build(nodes, 8)
	out := render(rows)
	t.Logf("\n%s", out)
	want := "" +
		"○─╮\n" +
		"│ │ ●\n" +
		"● │ │\n" +
		"│ ● │\n" +
		"│ ● │\n" +
		"●─╯─╯\n" +
		"●\n"
	if out != want {
		t.Fatalf("unexpected layout:\n%s\nwant:\n%s", out, want)
	}
}
