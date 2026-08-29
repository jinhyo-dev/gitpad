package git

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ExtractBlob writes the version of path at commit hash to a temporary file
// that keeps the original file name (so the OS picks the right application)
// and returns its location.
func (r *Runner) ExtractBlob(hash, path string) (string, error) {
	data, err := r.RunBytes("show", hash+":"+path)
	if err != nil {
		return "", err
	}
	dir := filepath.Join(os.TempDir(), "gitpad-preview", shortHash(hash))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	out := filepath.Join(dir, filepath.Base(path))
	if err := os.WriteFile(out, data, 0o644); err != nil {
		return "", err
	}
	return out, nil
}

// RunBytes is Run for binary output (images, PDFs…).
func (r *Runner) RunBytes(args ...string) ([]byte, error) {
	out, err := r.run(false, args...)
	if err != nil {
		return nil, err
	}
	return []byte(out), nil
}

// WorktreePath returns the absolute path of a working-tree file, or an
// error when it does not exist (e.g. a deleted file).
func (r *Runner) WorktreePath(path string) (string, error) {
	p := filepath.Join(r.Dir, path)
	if _, err := os.Stat(p); err != nil {
		return "", fmt.Errorf("%s is not in the working tree", strings.TrimSpace(path))
	}
	return p, nil
}
