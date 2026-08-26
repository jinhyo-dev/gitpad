// gitpad — a Git log & branch manager for the terminal.
package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/jinhyo/gitpad/internal/ui"
)

var version = "dev"

func main() {
	path := "."
	snapshot := ""
	for i := 1; i < len(os.Args); i++ {
		switch a := os.Args[i]; {
		case a == "-v" || a == "--version":
			fmt.Println("gitpad", version)
			return
		case a == "-h" || a == "--help":
			fmt.Println("usage: gitpad [path]\n\nOpens a Git log / branch manager for the repository at path (default: current directory).")
			return
		case strings.HasPrefix(a, "--snapshot="):
			snapshot = strings.TrimPrefix(a, "--snapshot=")
		default:
			path = a
		}
	}
	if snapshot != "" {
		renderSnapshot(path, snapshot)
		return
	}
	p := tea.NewProgram(ui.New(path), tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "gitpad:", err)
		os.Exit(1)
	}
}

// renderSnapshot renders one frame at WxH and prints it (used for debugging
// layout without a TTY).
func renderSnapshot(path, size string) {
	wh := strings.SplitN(size, "x", 2)
	w, h := 160, 45
	if len(wh) == 2 {
		w, _ = strconv.Atoi(wh[0])
		h, _ = strconv.Atoi(wh[1])
	}
	fmt.Print(ui.Snapshot(path, w, h))
}
