package ci

import "testing"

func TestParseRemote(t *testing.T) {
	cases := map[string]Remote{
		"https://github.com/jinhyo-dev/gitpad.git": {"github.com", "jinhyo-dev", "gitpad"},
		"https://github.com/jinhyo-dev/gitpad":     {"github.com", "jinhyo-dev", "gitpad"},
		"git@github.com:jinhyo-dev/gitpad.git":     {"github.com", "jinhyo-dev", "gitpad"},
		"ssh://git@github.com/jinhyo-dev/gitpad":   {"github.com", "jinhyo-dev", "gitpad"},
		"ssh://git@ghe.acme.dev:2222/team/app.git": {"ghe.acme.dev", "team", "app"},
	}
	for in, want := range cases {
		got, ok := ParseRemote(in)
		if !ok {
			t.Errorf("%s: not parsed", in)
			continue
		}
		if got != want {
			t.Errorf("%s: got %+v want %+v", in, got, want)
		}
	}
	if _, ok := ParseRemote("/local/path"); ok {
		t.Error("local path should not parse")
	}
}

func TestParseGitHubCommit(t *testing.T) {
	raw := []byte(`{"statusCheckRollup":{"state":"FAILURE","contexts":{"nodes":[
		{"__typename":"CheckRun","name":"test (ubuntu)","status":"COMPLETED","conclusion":"SUCCESS","detailsUrl":"u1","startedAt":"2026-08-26T10:00:00Z","completedAt":"2026-08-26T10:01:12Z"},
		{"__typename":"CheckRun","name":"release","status":"IN_PROGRESS","conclusion":null},
		{"__typename":"StatusContext","context":"ci/jenkins","state":"FAILURE","targetUrl":"u3"}]}}}`)
	r := ParseGitHubCommit(raw)
	if r.State != StateFailure || len(r.Checks) != 3 {
		t.Fatalf("unexpected result: %+v", r)
	}
	if r.Checks[0].State != StateSuccess || r.Checks[0].Duration != "1m 12s" {
		t.Errorf("check 0: %+v", r.Checks[0])
	}
	if r.Checks[1].State != StatePending {
		t.Errorf("check 1: %+v", r.Checks[1])
	}
	if r.Checks[2].State != StateFailure || r.Checks[2].Name != "ci/jenkins" {
		t.Errorf("check 2: %+v", r.Checks[2])
	}
	if ParseGitHubCommit([]byte(`{"statusCheckRollup":null}`)).State != StateNone {
		t.Error("null rollup should be StateNone")
	}
}
