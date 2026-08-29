package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// The test binary stands in for gitpad as GIT_SEQUENCE_EDITOR: TestMain
// implements the --write-todo helper when invoked that way.
func TestMain(m *testing.M) {
	if len(os.Args) == 4 && os.Args[1] == WriteTodoArg {
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

func TestInteractiveRebase(t *testing.T) {
	dir := t.TempDir()
	run(t, dir, "init", "-q", "-b", "main")
	for i, name := range []string{"one", "two", "three", "four"} {
		os.WriteFile(filepath.Join(dir, name+".txt"), []byte(name+"\n"), 0o644)
		run(t, dir, "add", ".")
		run(t, dir, "commit", "-q", "-m", "commit "+name)
		_ = i
	}
	r, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	log, _ := r.Log(LogOptions{})
	// log: four, three, two, one (newest first)
	plan, err := r.PlanRebase(log[2].Hash) // from "two"
	if err != nil {
		t.Fatal(err)
	}
	if plan.Root || plan.Base != log[3].Hash || len(plan.Commits) != 3 {
		t.Fatalf("unexpected plan: %+v", plan)
	}
	// Oldest first: reword "two", drop "three", keep "four".
	steps := []RebaseStep{
		{Hash: log[2].Hash, Subject: log[2].Subject, Action: Reword, Message: "commit TWO (reworded)"},
		{Hash: log[1].Hash, Subject: log[1].Subject, Action: Drop},
		{Hash: log[0].Hash, Subject: log[0].Subject, Action: Pick},
	}
	if err := r.RebaseInteractive(plan, steps); err != nil {
		t.Fatal(err)
	}
	out := strings.TrimSpace(run(t, dir, "log", "--format=%s"))
	want := "commit four\ncommit TWO (reworded)\ncommit one"
	if out != want {
		t.Fatalf("history after rebase:\n%s\nwant:\n%s", out, want)
	}
	if _, err := os.Stat(filepath.Join(dir, "three.txt")); !os.IsNotExist(err) {
		t.Fatal("dropped commit's file should be gone")
	}
	// Squash + reorder: put "four" before "TWO" and squash TWO into it.
	log, _ = r.Log(LogOptions{})
	plan, _ = r.PlanRebase(log[1].Hash)
	steps = []RebaseStep{
		{Hash: log[0].Hash, Subject: log[0].Subject, Action: Pick},
		{Hash: log[1].Hash, Subject: log[1].Subject, Action: Squash},
	}
	if err := r.RebaseInteractive(plan, steps); err != nil {
		t.Fatal(err)
	}
	out = strings.TrimSpace(run(t, dir, "log", "--format=%s"))
	if !strings.HasPrefix(out, "commit four") || strings.Count(out, "\n") != 1 {
		t.Fatalf("squash/reorder result:\n%s", out)
	}
	// Dirty tree is refused up front.
	os.WriteFile(filepath.Join(dir, "one.txt"), []byte("dirty\n"), 0o644)
	if _, err := r.PlanRebase(log[0].Hash); err == nil {
		t.Fatal("dirty working tree must be refused")
	}
	_ = exec.Command
}
