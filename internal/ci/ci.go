// Package ci fetches commit build/check status from code hosting providers so
// the log can show ✓ / ✗ / in-progress markers next to each commit.
package ci

import (
	"context"
	"net/url"
	"regexp"
	"strings"
)

// State is the aggregated status of a commit.
type State int

const (
	StateNone State = iota // no checks reported
	StatePending
	StateSuccess
	StateFailure
)

// Check is one individual check run or status context.
type Check struct {
	Name     string
	State    State
	Duration string // human readable, may be empty
	URL      string
}

// Result is the status of a single commit.
type Result struct {
	State  State
	Checks []Check
}

// Provider fetches results for a batch of commit hashes.
type Provider interface {
	Name() string
	// Fetch returns a result for every requested hash (missing = StateNone).
	Fetch(ctx context.Context, shas []string) (map[string]Result, error)
	// ChecksURL is the web page listing the checks of a commit.
	ChecksURL(sha string) string
}

// Remote describes a parsed remote URL.
type Remote struct {
	Host  string
	Owner string
	Repo  string
}

var sshRe = regexp.MustCompile(`^(?:ssh://)?(?:[\w.-]+@)?([\w.-]+)[:/]([\w.-]+)/([\w.-]+?)(?:\.git)?/?$`)

// ParseRemote understands https://host/owner/repo(.git), git@host:owner/repo(.git)
// and ssh://git@host/owner/repo forms.
func ParseRemote(raw string) (Remote, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return Remote{}, false
	}
	if strings.Contains(raw, "://") {
		u, err := url.Parse(raw)
		if err != nil {
			return Remote{}, false
		}
		parts := strings.Split(strings.Trim(u.Path, "/"), "/")
		if len(parts) < 2 || u.Hostname() == "" {
			return Remote{}, false
		}
		return Remote{Host: u.Hostname(), Owner: parts[0], Repo: strings.TrimSuffix(parts[1], ".git")}, true
	}
	m := sshRe.FindStringSubmatch(raw)
	if m == nil {
		return Remote{}, false
	}
	return Remote{Host: m[1], Owner: m[2], Repo: m[3]}, true
}

// Detect returns a provider for the remote, or nil when the host is not
// supported or no credentials are available.
func Detect(remoteURL string) Provider {
	r, ok := ParseRemote(remoteURL)
	if !ok {
		return nil
	}
	if r.Host == "github.com" || strings.Contains(r.Host, "github") {
		if p := newGitHub(r); p != nil {
			return p
		}
	}
	return nil
}
