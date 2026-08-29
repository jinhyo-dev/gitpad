package git

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const conflicted = `intro
<<<<<<< HEAD
ours line
=======
theirs line
>>>>>>> feature
middle
<<<<<<< HEAD
a
||||||| base
orig
=======
b
>>>>>>> feature
end
`

func TestParseAndResolve(t *testing.T) {
	lines, cs := ParseConflicts(conflicted)
	if len(cs) != 2 {
		t.Fatalf("expected 2 conflicts, got %d", len(cs))
	}
	if cs[0].OursLabel != "HEAD" || cs[0].TheirsLabel != "feature" || cs[0].Ours[0] != "ours line" || cs[0].Theirs[0] != "theirs line" {
		t.Fatalf("conflict 0 parsed wrong: %+v", cs[0])
	}
	if cs[1].Base == nil || cs[1].Base[0] != "orig" || cs[1].Ours[0] != "a" || cs[1].Theirs[0] != "b" {
		t.Fatalf("diff3 conflict parsed wrong: %+v", cs[1])
	}
	out, all := Resolve(lines, cs, []Choice{KeepTheirs, KeepBoth})
	if !all || out != "intro\ntheirs line\nmiddle\na\nb\nend\n" {
		t.Fatalf("resolve: all=%v\n%s", all, out)
	}
	out, all = Resolve(lines, cs, []Choice{KeepOurs, Unresolved})
	if all || !strings.Contains(out, "<<<<<<< HEAD\na\n") || strings.Contains(out, "theirs line") {
		t.Fatalf("partial resolve wrong: all=%v\n%s", all, out)
	}
}

func TestConflictedFilesAndWrite(t *testing.T) {
	dir := t.TempDir()
	run(t, dir, "init", "-q", "-b", "main")
	os.WriteFile(filepath.Join(dir, "f.txt"), []byte("one\ntwo\n"), 0o644)
	run(t, dir, "add", ".")
	run(t, dir, "commit", "-q", "-m", "base")
	run(t, dir, "checkout", "-q", "-b", "feature")
	os.WriteFile(filepath.Join(dir, "f.txt"), []byte("one\nTHEIRS\n"), 0o644)
	run(t, dir, "commit", "-q", "-am", "theirs")
	run(t, dir, "checkout", "-q", "main")
	os.WriteFile(filepath.Join(dir, "f.txt"), []byte("one\nOURS\n"), 0o644)
	run(t, dir, "commit", "-q", "-am", "ours")
	r, _ := Open(dir)
	if err := r.Merge("feature"); err == nil {
		t.Fatal("merge should conflict")
	}
	files, err := r.ConflictedFiles()
	if err != nil || len(files) != 1 || files[0] != "f.txt" {
		t.Fatalf("conflicted files: %v %v", files, err)
	}
	content, _ := r.ReadWorktreeFile("f.txt")
	lines, cs := ParseConflicts(content)
	out, all := Resolve(lines, cs, []Choice{KeepBoth})
	if !all {
		t.Fatal("should be resolved")
	}
	if err := r.WriteWorktreeFile("f.txt", out); err != nil {
		t.Fatal(err)
	}
	if err := r.StagePath("f.txt"); err != nil {
		t.Fatal(err)
	}
	if files, _ := r.ConflictedFiles(); len(files) != 0 {
		t.Fatalf("still conflicted: %v", files)
	}
	if err := r.ContinueState("merging"); err != nil {
		t.Fatal(err)
	}
	if got := run(t, dir, "show", "HEAD:f.txt"); got != "one\nOURS\nTHEIRS\n" {
		t.Fatalf("merged content: %q", got)
	}
}
