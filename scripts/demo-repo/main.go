// Command demo-repo builds a synthetic, fully anonymous git repository that
// looks like a busy product repo: ~400 commits over eight months, a dozen
// branches, PR-style merges, tags, an origin remote and a dirty working tree.
// It is used to produce the README screenshots reproducibly.
//
//	go run ./scripts/demo-repo /tmp/gitpad-demo
package main

import (
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type author struct{ name, email string }

var authors = []author{
	{"Alice Park", "alice@acme.dev"},
	{"Ben Cho", "ben@acme.dev"},
	{"Chloe Nguyen", "chloe@acme.dev"},
	{"Daniel Ruiz", "daniel@acme.dev"},
	{"Eunji Lee", "eunji@acme.dev"},
	{"acme-bot", "bot@acme.dev"},
}

var files = []string{
	"api/src/main/kotlin/com/acme/api/AuthController.kt",
	"api/src/main/kotlin/com/acme/api/BillingController.kt",
	"api/src/main/kotlin/com/acme/api/SearchService.kt",
	"api/src/main/kotlin/com/acme/api/UserRepository.kt",
	"api/src/main/kotlin/com/acme/api/RateLimiter.kt",
	"api/src/main/resources/application.yml",
	"api/src/test/kotlin/com/acme/api/AuthControllerTest.kt",
	"api/src/test/kotlin/com/acme/api/SearchServiceTest.kt",
	"api/build.gradle.kts",
	"web/src/app/App.tsx",
	"web/src/app/routes.tsx",
	"web/src/components/ProfileCard.tsx",
	"web/src/components/DataTable.tsx",
	"web/src/components/Sidebar.tsx",
	"web/src/hooks/useSession.ts",
	"web/src/api/client.ts",
	"web/src/styles/theme.css",
	"web/package.json",
	"infra/k8s/api-deployment.yaml",
	"infra/k8s/web-deployment.yaml",
	"infra/terraform/main.tf",
	".github/workflows/ci.yml",
	"docs/architecture.md",
	"docs/api/README.md",
	"README.md",
}

var subjects = []string{
	"feat(api): add rate limiter to public endpoints",
	"feat(api): expose search suggestions endpoint",
	"feat(web): sidebar collapse state persisted per user",
	"feat(web): keyboard shortcuts for the data table",
	"feat(auth): refresh tokens rotate on every use",
	"feat(billing): support annual plans with proration",
	"feat(billing): 결제 웹훅 재시도 로직 추가",
	"feat(web): 다크 모드 토글 및 시스템 테마 감지",
	"feat(api): 관리자 감사 로그 조회 API 추가",
	"feat(search): 검색 자동완성 캐시 적용",
	"fix(auth): handle expired session on redirect",
	"fix(web): null avatar in ProfileCard",
	"fix(api): off-by-one in cursor pagination",
	"fix(billing): 환불 금액 반올림 오류 수정",
	"fix(web): 로그인 만료 시 무한 리다이렉트 수정",
	"fix(api): N+1 query when listing organizations",
	"fix(infra): readiness probe path for the api pod",
	"fix(search): 한글 초성 검색 시 결과 누락",
	"fix(web): DataTable keeps stale sort after refetch",
	"perf(api): cache organization lookups for 60s",
	"perf(web): lazy-load the settings routes",
	"refactor(api): extract JWT verification into a filter",
	"refactor(web): move session logic into useSession",
	"refactor(api): 컨트롤러 인가 애노테이션 정리",
	"chore(deps): bump kotlin to 2.1.20",
	"chore(deps): bump vite to 6.3.2",
	"chore(ci): cache gradle wrapper between jobs",
	"chore: 배포 스크립트 정리",
	"docs: describe the rate limiting policy",
	"docs: 온보딩 가이드 업데이트",
	"test(api): regression test for cursor pagination",
	"test(web): cover ProfileCard empty states",
	"style(web): align table header with design tokens",
	"build: enable gradle configuration cache",
}

var branchSlugs = []string{
	"feat/rate-limiter", "feat/search-suggest", "feat/annual-billing", "feat/dark-mode",
	"feat/audit-log", "fix/session-redirect", "fix/pagination", "feat/keyboard-shortcuts",
	"refactor/jwt-filter", "chore/gradle-cache", "feat/webhook-retry", "fix/refund-rounding",
	"feat/org-cache", "feat/sso-login", "fix/avatar-fallback", "feat/export-csv",
	"chore/vite-6", "feat/team-invites", "fix/n-plus-one", "feat/usage-dashboard",
	"refactor/session-hook", "feat/api-keys", "fix/probe-path", "feat/notifications",
	"chore/kotlin-2.1", "feat/table-resize", "fix/timezone-offset", "feat/search-v2",
	"feat/#412-bulk-import", "fix/#398-empty-state", "feat/webhooks-ui", "fix/csv-encoding",
	"feat/rate-limit-headers", "chore/renovate", "feat/profile-avatars", "fix/sidebar-flicker",
	"feat/billing-portal", "refactor/search-index", "feat/audit-export", "fix/jwt-clock-skew",
	"feat/team-roles", "chore/gradle-8.12", "feat/onboarding-tour", "fix/probe-timeouts",
	"feat/i18n-ko", "feat/theme-tokens", "fix/table-virtualization", "chore/eslint-9",
}

var tags = []string{"v1.0.0", "v1.1.0", "v1.2.0", "v1.3.0", "v1.4.0", "v1.4.1"}

type repo struct {
	dir string
	rng *rand.Rand
	now time.Time
	pr  int
}

func (r *repo) git(env []string, args ...string) string {
	cmd := exec.Command("git", args...)
	cmd.Dir = r.dir
	cmd.Env = append(os.Environ(), env...)
	cmd.Env = append(cmd.Env, "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null")
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "git %s: %v\n%s", strings.Join(args, " "), err, out)
		os.Exit(1)
	}
	return string(out)
}

func (r *repo) advance() {
	r.now = r.now.Add(time.Duration(20+r.rng.Intn(160)) * time.Minute)
	if h := r.now.Hour(); h >= 20 || h < 9 { // skip the night
		r.now = time.Date(r.now.Year(), r.now.Month(), r.now.Day()+1, 9+r.rng.Intn(2), r.rng.Intn(60), 0, 0, r.now.Location())
	}
	if r.now.Weekday() == time.Saturday {
		r.now = r.now.Add(48 * time.Hour)
	}
}

// pool restricts which files a commit may touch, so that a feature branch and
// the concurrent commits on main never edit the same file (no merge conflicts).
func (r *repo) commit(a author, msg string) { r.commitIn(files, a, msg) }

func (r *repo) commitIn(pool []string, a author, msg string) {
	n := 1 + r.rng.Intn(3)
	for i := 0; i < n; i++ {
		f := pool[r.rng.Intn(len(pool))]
		p := filepath.Join(r.dir, f)
		os.MkdirAll(filepath.Dir(p), 0o755)
		fh, _ := os.OpenFile(p, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		fmt.Fprintf(fh, "// %s — %s\n", r.now.Format("2006-01-02 15:04"), strings.SplitN(msg, "\n", 2)[0])
		fh.Close()
	}
	r.advance()
	date := r.now.Format(time.RFC3339)
	env := []string{"GIT_AUTHOR_NAME=" + a.name, "GIT_AUTHOR_EMAIL=" + a.email, "GIT_AUTHOR_DATE=" + date,
		"GIT_COMMITTER_NAME=" + a.name, "GIT_COMMITTER_EMAIL=" + a.email, "GIT_COMMITTER_DATE=" + date}
	r.git(nil, "add", "-A")
	r.git(env, "commit", "-q", "--allow-empty", "-m", msg)
}

func (r *repo) merge(branch string, a author) {
	r.pr++
	r.advance()
	date := r.now.Format(time.RFC3339)
	env := []string{"GIT_AUTHOR_NAME=" + a.name, "GIT_AUTHOR_EMAIL=" + a.email, "GIT_AUTHOR_DATE=" + date,
		"GIT_COMMITTER_NAME=" + a.name, "GIT_COMMITTER_EMAIL=" + a.email, "GIT_COMMITTER_DATE=" + date}
	r.git(env, "merge", "-q", "--no-ff", "-m", fmt.Sprintf("Merge pull request #%d from acme/%s", 300+r.pr, branch), branch)
}

func (r *repo) subject() string {
	s := subjects[r.rng.Intn(len(subjects))]
	if r.rng.Intn(4) == 0 {
		s += "\n\nRefs ACME-" + fmt.Sprint(1000+r.rng.Intn(900)) + "\n"
	}
	return s
}

// splitPool gives a branch and main disjoint thirds of the file list.
func splitPool(bi int) (branch, main []string) {
	third := len(files) / 3
	a, b := bi%3, (bi+1)%3
	return files[a*third : (a+1)*third], files[b*third : (b+1)*third]
}

func (r *repo) author() author { return authors[r.rng.Intn(len(authors)-1)] }

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: demo-repo <dir>")
		os.Exit(2)
	}
	dir, _ := filepath.Abs(os.Args[1])
	os.RemoveAll(dir)
	os.RemoveAll(dir + "-origin.git")
	os.MkdirAll(dir, 0o755)
	r := &repo{dir: dir, rng: rand.New(rand.NewSource(20260826)), now: time.Date(2026, 1, 5, 9, 30, 0, 0, time.Local)}

	r.git(nil, "init", "-q", "-b", "main")
	os.WriteFile(filepath.Join(dir, "README.md"), []byte("# acme platform\n\nMonorepo for the Acme API, web app and infrastructure.\n"), 0o644)
	for _, f := range files {
		p := filepath.Join(dir, f)
		os.MkdirAll(filepath.Dir(p), 0o755)
		if _, err := os.Stat(p); err != nil {
			os.WriteFile(p, []byte("// "+filepath.Base(f)+"\n"), 0o644)
		}
	}
	r.commit(authors[0], "chore: bootstrap monorepo")
	r.commit(authors[1], "feat(api): initial Spring Boot skeleton")
	r.commit(authors[2], "feat(web): initial Vite + React skeleton")
	r.commit(authors[5], "chore(ci): add GitHub Actions workflow")
	r.git(nil, "branch", "develop")

	total := 4
	tagIdx := 0
	open := []string{}
	const target = 400
	for bi := 0; total < target-6; bi++ {
		// A few direct commits on main.
		for i := 0; i < 1+r.rng.Intn(5); i++ {
			r.commit(r.author(), r.subject())
			total++
		}
		branch := branchSlugs[bi%len(branchSlugs)]
		if bi >= len(branchSlugs) {
			branch = fmt.Sprintf("%s-%d", branch, bi/len(branchSlugs)+1)
		}
		owner := r.author()
		branchPool, mainPool := splitPool(bi)
		r.git(nil, "checkout", "-q", "-b", branch)
		n := 3 + r.rng.Intn(7)
		for i := 0; i < n; i++ {
			r.commitIn(branchPool, owner, r.subject())
			total++
			// Interleave work on main so lanes run in parallel.
			if r.rng.Intn(3) == 0 {
				r.git(nil, "checkout", "-q", "main")
				r.commitIn(mainPool, r.author(), r.subject())
				total++
				r.git(nil, "checkout", "-q", branch)
			}
		}
		r.git(nil, "checkout", "-q", "main")
		if total >= target-40 && len(open) < 3 { // leave the last few branches open
			open = append(open, branch)
			continue
		}
		r.merge(branch, authors[r.rng.Intn(2)])
		total++
		r.git(nil, "branch", "-q", "-D", branch)
		if bi%5 == 4 && tagIdx < len(tags) {
			r.git(nil, "tag", "-a", tags[tagIdx], "-m", "release "+tags[tagIdx])
			tagIdx++
		}
		if bi%3 == 2 { // keep develop trailing main
			r.git(nil, "branch", "-f", "develop", "HEAD~2")
		}
	}
	if tagIdx == 0 {
		r.git(nil, "tag", "-a", tags[0], "-m", "release "+tags[0])
		tagIdx = 1
	}
	r.git(nil, "branch", "staging", "HEAD~5")
	r.git(nil, "branch", "release/"+strings.TrimPrefix(tags[tagIdx-1], "v"), tags[tagIdx-1])

	// Remote: a bare origin with everything pushed, then diverge a little.
	r.git(nil, "init", "-q", "--bare", dir+"-origin.git")
	r.git(nil, "remote", "add", "origin", dir+"-origin.git")
	r.git(nil, "push", "-q", "--all", "origin")
	r.git(nil, "push", "-q", "--tags", "origin")
	r.git(nil, "branch", "-q", "--set-upstream-to=origin/main", "main")
	for _, b := range open {
		r.git(nil, "branch", "-q", "--set-upstream-to=origin/"+b, b)
	}
	r.git(nil, "branch", "-q", "-D", open[len(open)-1]) // remote-only branch
	r.commit(authors[0], "feat(api): MeloTTS-style streaming synthesis proxy (/api/v2/tts)")
	r.commit(authors[0], "fix(api): MANAGER role may read contents and widgets")

	// Dirty working tree.
	appendLine := func(f, line string) {
		p := filepath.Join(dir, f)
		fh, _ := os.OpenFile(p, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		fmt.Fprintln(fh, line)
		fh.Close()
	}
	appendLine("api/src/main/kotlin/com/acme/api/SearchService.kt", "// TODO: debounce suggestions on the client")
	appendLine("web/src/components/DataTable.tsx", "// keep column widths in localStorage")
	os.MkdirAll(filepath.Join(dir, "docs/adr"), 0o755)
	os.WriteFile(filepath.Join(dir, "docs/adr/0007-search-v2.md"), []byte("# ADR 7: search v2\n\nReplace the LIKE-based search with the indexed suggester.\n"), 0o644)
	os.WriteFile(filepath.Join(dir, ".env.local"), []byte("API_URL=http://localhost:8080\n"), 0o644)

	fmt.Printf("demo repository ready: %s (%d commits)\n", dir, total+2)
}
