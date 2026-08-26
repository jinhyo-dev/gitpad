// Package git wraps the git command line. gitpad deliberately shells out to the
// user's git binary instead of using a library so that hooks, credential
// helpers, GPG signing and every .gitconfig setting behave exactly as in the
// terminal.
package git

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Entry is a record of one executed git command (shown in the Console tab).
type Entry struct {
	Args   []string
	Output string
	Err    error
	Took   time.Duration
	At     time.Time
}

// Error is returned when git exits non-zero.
type Error struct {
	Args   []string
	Stderr string
	Code   int
}

func (e *Error) Error() string {
	msg := strings.TrimSpace(e.Stderr)
	if msg == "" {
		return "git " + strings.Join(e.Args, " ") + " failed"
	}
	// Only keep the most relevant line for toasts; full text is in the console.
	lines := strings.Split(msg, "\n")
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if strings.HasPrefix(l, "error:") || strings.HasPrefix(l, "fatal:") {
			return l
		}
	}
	return lines[len(lines)-1]
}

// Runner executes git commands inside a repository.
type Runner struct {
	Dir string // repository top-level

	mu      sync.Mutex
	history []Entry
}

// Open locates the repository containing path.
func Open(path string) (*Runner, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	cmd := exec.Command("git", "-C", abs, "rev-parse", "--show-toplevel")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return nil, errors.New(msg)
	}
	return &Runner{Dir: strings.TrimSpace(string(out))}, nil
}

// Run executes git with args and returns stdout.
func (r *Runner) Run(args ...string) (string, error) {
	return r.run(false, args...)
}

// RunWrite executes a mutating command (index locks enabled).
func (r *Runner) RunWrite(args ...string) (string, error) {
	return r.run(true, args...)
}

func (r *Runner) run(write bool, args ...string) (string, error) {
	start := time.Now()
	full := append([]string{"--no-pager"}, args...)
	cmd := exec.Command("git", full...)
	cmd.Dir = r.Dir
	env := append(os.Environ(), "GIT_TERMINAL_PROMPT=0", "GIT_EDITOR=true", "GIT_SEQUENCE_EDITOR=true")
	if !write {
		env = append(env, "GIT_OPTIONAL_LOCKS=0")
	}
	cmd.Env = env
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	took := time.Since(start)

	var result error
	if err != nil {
		code := -1
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			code = ee.ExitCode()
		}
		result = &Error{Args: args, Stderr: stderr.String(), Code: code}
		if stderr.Len() == 0 {
			result = &Error{Args: args, Stderr: err.Error(), Code: code}
		}
	}
	output := stdout.String()
	logOut := output
	if stderr.Len() > 0 {
		if logOut != "" && !strings.HasSuffix(logOut, "\n") {
			logOut += "\n"
		}
		logOut += stderr.String()
	}
	r.mu.Lock()
	r.history = append(r.history, Entry{Args: args, Output: logOut, Err: result, Took: took, At: start})
	if len(r.history) > 500 {
		r.history = r.history[len(r.history)-500:]
	}
	r.mu.Unlock()
	return output, result
}

// History returns a copy of the executed command log, newest last.
func (r *Runner) History() []Entry {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]Entry, len(r.history))
	copy(out, r.history)
	return out
}

// Name is the repository directory name.
func (r *Runner) Name() string {
	return filepath.Base(r.Dir)
}
