package ui

import (
	"os/exec"
	"path/filepath"
	"runtime"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/jinhyo-dev/gitpad/internal/git"
)

// Opening files with the desktop: images, PDFs, office documents and
// anything else the terminal cannot show get handed to the default app.

type openedMsg struct {
	what string
	err  error
}

// openWithDefaultApp opens a file or URL with the platform handler.
func openWithDefaultApp(target string) *exec.Cmd {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", target)
	case "windows":
		return exec.Command("cmd", "/c", "start", "", target)
	default:
		return exec.Command("xdg-open", target)
	}
}

// revealInFileManager shows the file in Finder / Explorer / the file manager.
func revealInFileManager(path string) *exec.Cmd {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", "-R", path)
	case "windows":
		return exec.Command("explorer", "/select,", path)
	default:
		return exec.Command("xdg-open", filepath.Dir(path))
	}
}

// previewFile opens the selected file with its default application: the
// working-tree file for local changes, otherwise that commit's version.
func (m *Model) previewFile(fc git.FileChange, reveal bool) tea.Cmd {
	repo := m.repo
	local := m.filesFor == "local"
	var commit *git.Commit
	if !local {
		commit = m.selectedCommit()
	}
	if fc.Status == 'D' {
		return m.showToast("Deleted files have nothing to open", 2)
	}
	return func() tea.Msg {
		var path string
		var err error
		switch {
		case local || commit == nil:
			path, err = repo.WorktreePath(fc.Path)
		case reveal:
			path, err = repo.WorktreePath(fc.Path) // reveal only makes sense on disk
		default:
			path, err = repo.ExtractBlob(commit.Hash, fc.Path)
		}
		if err != nil {
			return openedMsg{what: fc.Path, err: err}
		}
		cmd := openWithDefaultApp(path)
		if reveal {
			cmd = revealInFileManager(path)
		}
		if err := cmd.Start(); err != nil {
			return openedMsg{what: fc.Path, err: err}
		}
		what := "Opened " + filepath.Base(fc.Path)
		if reveal {
			what = "Revealed " + filepath.Base(fc.Path)
		} else if commit != nil {
			what += " @ " + commit.Short
		}
		return openedMsg{what: what}
	}
}
