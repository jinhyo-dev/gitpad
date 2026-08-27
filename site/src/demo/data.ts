// A deterministic, fictional repository for the browser demo.
export type Author = { name: string; email: string };
export type Ref = { name: string; kind: "head" | "local" | "remote" | "tag" };
export type Commit = {
  hash: string;
  parents: string[];
  author: Author;
  when: Date;
  subject: string;
  body?: string;
  refs: Ref[];
  files: FileChange[];
  ci?: "ok" | "fail" | "pending";
};
export type FileChange = { status: "M" | "A" | "D" | "?"; path: string; diff: string };
export type Branch = {
  name: string;
  hash: string;
  kind: "local" | "remote" | "tag";
  ahead?: number;
};

export const authors: Author[] = [
  { name: "Alice Park", email: "alice@acme.dev" },
  { name: "Ben Cho", email: "ben@acme.dev" },
  { name: "Chloe Nguyen", email: "chloe@acme.dev" },
  { name: "Daniel Ruiz", email: "daniel@acme.dev" },
  { name: "Eunji Lee", email: "eunji@acme.dev" },
];

const subjects = [
  "feat(api): add rate limiter to public endpoints",
  "fix(web): null avatar in ProfileCard",
  "feat(billing): 결제 웹훅 재시도 로직 추가",
  "refactor(api): extract JWT verification into a filter",
  "fix(search): 한글 초성 검색 시 결과 누락",
  "perf(web): lazy-load the settings routes",
  "chore(deps): bump vite to 6.3.2",
  "docs: 온보딩 가이드 업데이트",
  "test(api): regression test for cursor pagination",
  "feat(web): 다크 모드 토글 및 시스템 테마 감지",
  "fix(api): N+1 query when listing organizations",
  "feat(search): 검색 자동완성 캐시 적용",
  "style(web): align table header with design tokens",
  "fix(infra): readiness probe path for the api pod",
  "feat(auth): refresh tokens rotate on every use",
  "chore: 배포 스크립트 정리",
];

const paths = [
  "api/src/main/kotlin/com/acme/api/AuthController.kt",
  "api/src/main/kotlin/com/acme/api/SearchService.kt",
  "api/src/main/resources/application.yml",
  "web/src/components/ProfileCard.tsx",
  "web/src/components/DataTable.tsx",
  "web/src/hooks/useSession.ts",
  "infra/k8s/api-deployment.yaml",
  "docs/architecture.md",
];

// Small seeded PRNG so every visitor sees the same repository.
function rng(seed: number) {
  let s = seed >>> 0;
  return () => {
    s = (s * 1664525 + 1013904223) >>> 0;
    return s / 0x100000000;
  };
}

function fakeHash(r: () => number): string {
  let h = "";
  for (let i = 0; i < 40; i++) h += "0123456789abcdef"[Math.floor(r() * 16)];
  return h;
}

function fakeDiff(path: string, subject: string, r: () => number): string {
  const line = Math.floor(r() * 80) + 10;
  const ext = path.split(".").pop();
  const ctx =
    ext === "kt"
      ? "    fun handle(request: Request): Response {"
      : ext === "tsx" || ext === "ts"
        ? "  const { user } = useSession();"
        : "  replicas: 2";
  return [
    `diff --git a/${path} b/${path}`,
    `--- a/${path}`,
    `+++ b/${path}`,
    `@@ -${line},4 +${line},5 @@`,
    ` ${ctx}`,
    `-    // TODO`,
    `+    // ${subject}`,
    `+    log.debug("handled")`,
    ` }`,
  ].join("\n");
}

export type Repo = { commits: Commit[]; branches: Branch[]; head: string; local: FileChange[] };

export function makeRepo(): Repo {
  const r = rng(20260826);
  const commits: Commit[] = [];
  let when = new Date(2026, 7, 21, 11, 31).getTime();
  const step = () => {
    when -= (30 + Math.floor(r() * 300)) * 60 * 1000;
    return new Date(when);
  };
  const files = (subject: string, n: number): FileChange[] =>
    Array.from({ length: n }, (_, i) => {
      const p = paths[(Math.floor(r() * paths.length) + i) % paths.length];
      return { status: r() < 0.15 ? "A" : "M", path: p, diff: fakeDiff(p, subject, r) };
    });
  const mk = (
    subject: string,
    parents: string[],
    a: Author,
    extra: Partial<Commit> = {},
  ): Commit => {
    const c: Commit = {
      hash: fakeHash(r),
      parents,
      author: a,
      when: step(),
      subject,
      refs: [],
      files: files(subject, 1 + Math.floor(r() * 3)),
      ci: r() < 0.85 ? "ok" : r() < 0.5 ? "fail" : "pending",
      ...extra,
    };
    commits.push(c);
    return c;
  };

  // Built newest → oldest; hashes are allocated as we go and parents are
  // filled in through placeholders.
  const H = (i: number) => `#${i}`; // placeholder resolved below
  const plan: {
    subject: string;
    parents: number[];
    author: number;
    refs?: Ref[];
    ci?: Commit["ci"];
  }[] = [
    {
      subject: "feat(api): MeloTTS-style streaming synthesis proxy (/api/v2/tts)",
      parents: [1],
      author: 0,
      refs: [
        { name: "HEAD", kind: "head" },
        { name: "main", kind: "local" },
      ],
      ci: "pending",
    },
    {
      subject: "fix(api): MANAGER role may read contents and widgets",
      parents: [2],
      author: 0,
      ci: "ok",
    },
    {
      subject: "Merge pull request #341 from acme/feat/search-suggest",
      parents: [4, 3],
      author: 1,
      refs: [{ name: "origin/main", kind: "remote" }],
    },
    { subject: "feat(search): 검색 자동완성 캐시 적용", parents: [5], author: 3 },
    { subject: "fix(api): off-by-one in cursor pagination", parents: [6], author: 0 },
    { subject: "perf(api): cache organization lookups for 60s", parents: [7], author: 3 },
    { subject: "refactor(api): 컨트롤러 인가 애노테이션 정리", parents: [8], author: 2 },
    { subject: "test(web): cover ProfileCard empty states", parents: [8], author: 3 },
    {
      subject: "Merge pull request #340 from acme/feat/dark-mode",
      parents: [10, 9],
      author: 1,
      refs: [{ name: "v1.4.0", kind: "tag" }],
    },
    {
      subject: "feat(web): 다크 모드 토글 및 시스템 테마 감지",
      parents: [11],
      author: 4,
      refs: [{ name: "origin/feat/dark-mode", kind: "remote" }],
    },
    { subject: "chore(deps): bump kotlin to 2.1.20", parents: [12], author: 1 },
    { subject: "feat(web): keyboard shortcuts for the data table", parents: [12], author: 4 },
    {
      subject: "fix(billing): 환불 금액 반올림 오류 수정",
      parents: [13],
      author: 2,
      refs: [{ name: "develop", kind: "local" }],
    },
    {
      subject: "Merge pull request #339 from acme/fix/refund-rounding",
      parents: [15, 14],
      author: 1,
    },
    { subject: "fix(billing): 환불 금액 반올림 오류 수정 (side)", parents: [16], author: 2 },
    { subject: "docs: 온보딩 가이드 업데이트", parents: [16], author: 0 },
    {
      subject: "build: enable gradle configuration cache",
      parents: [17],
      author: 4,
      refs: [
        { name: "staging", kind: "local" },
        { name: "origin/staging", kind: "remote" },
      ],
    },
  ];
  // Pad with older linear history.
  for (let i = plan.length; i < 60; i++) {
    plan.push({
      subject: subjects[Math.floor(r() * subjects.length)],
      parents: i < 59 ? [i + 1] : [],
      author: Math.floor(r() * authors.length),
    });
  }
  for (const p of plan)
    mk(p.subject, p.parents.map(H), authors[p.author], {
      refs: p.refs ?? [],
      ci: p.ci ?? (r() < 0.85 ? "ok" : "fail"),
    });
  for (const c of commits)
    c.parents = c.parents.map((p) => commits[Number(p.slice(1))]?.hash ?? "").filter(Boolean);

  const find = (name: string) => commits.find((c) => c.refs.some((x) => x.name === name))!.hash;
  const branches: Branch[] = [
    { name: "main", hash: commits[0].hash, kind: "local", ahead: 2 },
    { name: "develop", hash: find("develop"), kind: "local" },
    { name: "staging", hash: find("staging"), kind: "local" },
    { name: "origin/main", hash: find("origin/main"), kind: "remote" },
    { name: "origin/feat/dark-mode", hash: find("origin/feat/dark-mode"), kind: "remote" },
    { name: "origin/staging", hash: find("origin/staging"), kind: "remote" },
    { name: "v1.4.0", hash: find("v1.4.0"), kind: "tag" },
  ];
  const local: FileChange[] = [
    {
      status: "M",
      path: "api/src/main/kotlin/com/acme/api/SearchService.kt",
      diff: fakeDiff(
        "api/src/main/kotlin/com/acme/api/SearchService.kt",
        "debounce suggestions on the client",
        r,
      ),
    },
    {
      status: "M",
      path: "web/src/components/DataTable.tsx",
      diff: fakeDiff("web/src/components/DataTable.tsx", "keep column widths in localStorage", r),
    },
    {
      status: "?",
      path: "docs/adr/0007-search-v2.md",
      diff: "diff --git a/docs/adr/0007-search-v2.md b/docs/adr/0007-search-v2.md\n--- /dev/null\n+++ b/docs/adr/0007-search-v2.md\n@@ -0,0 +1,3 @@\n+# ADR 7: search v2\n+\n+Replace the LIKE-based search with the indexed suggester.",
    },
    {
      status: "?",
      path: ".env.local",
      diff: "diff --git a/.env.local b/.env.local\n--- /dev/null\n+++ b/.env.local\n@@ -0,0 +1 @@\n+API_URL=http://localhost:8080",
    },
  ];
  return { commits, branches, head: commits[0].hash, local };
}
