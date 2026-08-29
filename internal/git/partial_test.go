package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func run(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=T", "GIT_AUTHOR_EMAIL=t@x", "GIT_COMMITTER_NAME=T", "GIT_COMMITTER_EMAIL=t@x", "GIT_CONFIG_GLOBAL=/dev/null")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return string(out)
}

func lines(n int) string {
	var b strings.Builder
	for i := 1; i <= n; i++ {
		b.WriteString("line " + strings.Repeat("x", i%3) + "\n")
	}
	return b.String()
}

func TestFilterHunks(t *testing.T) {
	diff := "diff --git a/f b/f\n--- a/f\n+++ b/f\n@@ -1,2 +1,2 @@\n-a\n+A\n b\n@@ -10,2 +10,2 @@\n-j\n+J\n k\n"
	if HunkCount(diff) != 2 {
		t.Fatal("hunk count")
	}
	got := FilterHunks(diff, []int{1})
	if strings.Contains(got, "+A") || !strings.Contains(got, "+J") || !strings.HasPrefix(got, "diff --git") {
		t.Fatalf("unexpected patch:\n%s", got)
	}
	if FilterHunks(diff, nil) != "" {
		t.Fatal("no hunks should give an empty patch")
	}
}

func TestCommitSelectionPartial(t *testing.T) {
	dir := t.TempDir()
	run(t, dir, "init", "-q", "-b", "main")
	base := lines(40)
	os.WriteFile(filepath.Join(dir, "f.txt"), []byte(base), 0o644)
	os.WriteFile(filepath.Join(dir, "other.txt"), []byte("o\n"), 0o644)
	run(t, dir, "add", ".")
	run(t, dir, "commit", "-q", "-m", "base")

	// Two far-apart edits → two hunks; plus an unrelated dirty file that is
	// staged but must NOT be committed.
	ls := strings.Split(strings.TrimRight(base, "\n"), "\n")
	ls[1] = "EDIT-TOP"
	ls[37] = "EDIT-BOTTOM"
	os.WriteFile(filepath.Join(dir, "f.txt"), []byte(strings.Join(ls, "\n")+"\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "other.txt"), []byte("changed\n"), 0o644)
	run(t, dir, "add", "other.txt")

	r, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	diff, _ := r.Run("diff", "HEAD", "--", "f.txt")
	if HunkCount(diff) != 2 {
		t.Fatalf("expected 2 hunks, got %d:\n%s", HunkCount(diff), diff)
	}
	if err := r.CommitSelection("partial", []Selection{{Path: "f.txt", Hunks: []int{0}}}); err != nil {
		t.Fatal(err)
	}
	head := run(t, dir, "show", "HEAD:f.txt")
	if !strings.Contains(head, "EDIT-TOP") || strings.Contains(head, "EDIT-BOTTOM") {
		t.Fatalf("HEAD should contain only the first hunk:\n%s", head)
	}
	if other := run(t, dir, "show", "HEAD:other.txt"); other != "o\n" {
		t.Fatalf("other.txt must not be committed, got %q", other)
	}
	// Working tree still has both edits; status shows only the remainder.
	wt, _ := os.ReadFile(filepath.Join(dir, "f.txt"))
	if !strings.Contains(string(wt), "EDIT-BOTTOM") {
		t.Fatal("working tree lost the unselected hunk")
	}
	st := run(t, dir, "status", "--porcelain")
	if !strings.Contains(st, " M f.txt") || !strings.Contains(st, "other.txt") {
		t.Fatalf("unexpected status:\n%s", st)
	}
	rest, _ := r.Run("diff", "HEAD", "--", "f.txt")
	if HunkCount(rest) != 1 || !strings.Contains(rest, "EDIT-BOTTOM") {
		t.Fatalf("remaining diff should be the bottom hunk only:\n%s", rest)
	}

	// Failure restores the index: an empty message must not change anything.
	before := run(t, dir, "status", "--porcelain")
	if err := r.CommitSelection("", []Selection{{Path: "f.txt"}}); err == nil {
		t.Fatal("empty message must fail")
	}
	if after := run(t, dir, "status", "--porcelain"); after != before {
		t.Fatalf("index not restored:\n%s\nvs\n%s", before, after)
	}
}
