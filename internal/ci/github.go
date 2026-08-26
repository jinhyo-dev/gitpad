package ci

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

// GitHub fetches statusCheckRollup for many commits in one GraphQL request.
type GitHub struct {
	remote   Remote
	token    string
	endpoint string
	client   *http.Client
}

func newGitHub(r Remote) *GitHub {
	token := githubToken(r.Host)
	if token == "" {
		return nil
	}
	endpoint := "https://api.github.com/graphql"
	if r.Host != "github.com" {
		endpoint = "https://" + r.Host + "/api/graphql"
	}
	return &GitHub{remote: r, token: token, endpoint: endpoint, client: &http.Client{Timeout: 12 * time.Second}}
}

// githubToken looks for a token in the environment, then asks the gh CLI.
func githubToken(host string) string {
	for _, k := range []string{"GH_TOKEN", "GITHUB_TOKEN"} {
		if v := strings.TrimSpace(os.Getenv(k)); v != "" {
			return v
		}
	}
	if _, err := exec.LookPath("gh"); err != nil {
		return ""
	}
	out, err := exec.Command("gh", "auth", "token", "--hostname", host).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func (g *GitHub) Name() string { return "GitHub" }

func (g *GitHub) ChecksURL(sha string) string {
	return fmt.Sprintf("https://%s/%s/%s/commit/%s/checks", g.remote.Host, g.remote.Owner, g.remote.Repo, sha)
}

const batchSize = 50

func (g *GitHub) Fetch(ctx context.Context, shas []string) (map[string]Result, error) {
	out := make(map[string]Result, len(shas))
	for i := 0; i < len(shas); i += batchSize {
		end := i + batchSize
		if end > len(shas) {
			end = len(shas)
		}
		if err := g.fetchBatch(ctx, shas[i:end], out); err != nil {
			return out, err
		}
	}
	return out, nil
}

func (g *GitHub) fetchBatch(ctx context.Context, shas []string, out map[string]Result) error {
	var q strings.Builder
	fmt.Fprintf(&q, "query { rateLimit { remaining } repository(owner: %q, name: %q) {", g.remote.Owner, g.remote.Repo)
	for i, sha := range shas {
		fmt.Fprintf(&q, ` c%d: object(oid: %q) { ... on Commit { statusCheckRollup { state contexts(first: 50) { nodes {
  __typename
  ... on CheckRun { name status conclusion detailsUrl startedAt completedAt }
  ... on StatusContext { context state targetUrl }
} } } } }`, i, sha)
	}
	q.WriteString(" } }")
	body, _ := json.Marshal(map[string]string{"query": q.String()})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, g.endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+g.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "gitpad")
	resp, err := g.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("github api: %s", resp.Status)
	}
	var payload struct {
		Data   map[string]json.RawMessage `json:"data"`
		Errors []struct{ Message string } `json:"errors"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return err
	}
	if len(payload.Errors) > 0 && payload.Data == nil {
		return fmt.Errorf("github api: %s", payload.Errors[0].Message)
	}
	var repo map[string]json.RawMessage
	if raw, ok := payload.Data["repository"]; ok && string(raw) != "null" {
		if err := json.Unmarshal(raw, &repo); err != nil {
			return err
		}
	}
	for i, sha := range shas {
		out[sha] = parseCommit(repo[fmt.Sprintf("c%d", i)])
	}
	return nil
}

// parseCommit converts one commit object into a Result. It is exported for
// tests through ParseGitHubCommit.
func parseCommit(raw json.RawMessage) Result {
	if len(raw) == 0 || string(raw) == "null" {
		return Result{}
	}
	var c struct {
		Rollup *struct {
			State    string `json:"state"`
			Contexts struct {
				Nodes []struct {
					Typename    string `json:"__typename"`
					Name        string `json:"name"`
					Status      string `json:"status"`
					Conclusion  string `json:"conclusion"`
					DetailsURL  string `json:"detailsUrl"`
					StartedAt   string `json:"startedAt"`
					CompletedAt string `json:"completedAt"`
					Context     string `json:"context"`
					State       string `json:"state"`
					TargetURL   string `json:"targetUrl"`
				} `json:"nodes"`
			} `json:"contexts"`
		} `json:"statusCheckRollup"`
	}
	if err := json.Unmarshal(raw, &c); err != nil || c.Rollup == nil {
		return Result{}
	}
	res := Result{State: rollupState(c.Rollup.State)}
	for _, n := range c.Rollup.Contexts.Nodes {
		if n.Typename == "CheckRun" {
			ch := Check{Name: n.Name, URL: n.DetailsURL}
			switch {
			case n.Status != "COMPLETED":
				ch.State = StatePending
			case n.Conclusion == "SUCCESS" || n.Conclusion == "NEUTRAL" || n.Conclusion == "SKIPPED":
				ch.State = StateSuccess
			default:
				ch.State = StateFailure
			}
			if s, e := parseTime(n.StartedAt), parseTime(n.CompletedAt); !s.IsZero() && !e.IsZero() {
				ch.Duration = fmtDuration(e.Sub(s))
			}
			res.Checks = append(res.Checks, ch)
		} else {
			res.Checks = append(res.Checks, Check{Name: n.Context, State: rollupState(n.State), URL: n.TargetURL})
		}
	}
	return res
}

// ParseGitHubCommit exposes the response parser for tests.
func ParseGitHubCommit(raw []byte) Result { return parseCommit(raw) }

func rollupState(s string) State {
	switch s {
	case "SUCCESS":
		return StateSuccess
	case "FAILURE", "ERROR":
		return StateFailure
	case "PENDING", "EXPECTED":
		return StatePending
	}
	return StateNone
}

func parseTime(s string) time.Time {
	t, _ := time.Parse(time.RFC3339, s)
	return t
}

func fmtDuration(d time.Duration) string {
	d = d.Round(time.Second)
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	return fmt.Sprintf("%dm %02ds", int(d.Minutes()), int(d.Seconds())%60)
}
