package ui

import (
	"os"
	"testing"

	"github.com/jinhyo-dev/gitpad/internal/git"
)

// The test binary is what os.Executable() returns during tests, so it must
// provide the --write-todo helper that main.go implements for rebases.
func TestMain(m *testing.M) {
	if len(os.Args) == 4 && os.Args[1] == git.WriteTodoArg {
		data, err := os.ReadFile(os.Args[2])
		if err == nil {
			err = os.WriteFile(os.Args[3], data, 0o600)
		}
		if err != nil {
			os.Exit(1)
		}
		os.Exit(0)
	}
	os.Exit(m.Run())
}
