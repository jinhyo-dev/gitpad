export const en = {
  locale: "en",
  meta: {
    title: "gitpad · Git TUI: commit graph, branches & push in the terminal",
    description:
      "gitpad is an open-source Git TUI: a terminal client with a commit graph, branch tree, diffs, commit workspace, push dialog and GitHub CI status. One binary for macOS, Linux and Windows.",
    keywords: [
      "gitpad",
      "git tui",
      "git terminal",
      "github terminal",
      "terminal git client",
      "git log graph",
      "git branch manager",
      "git cli ui",
      "terminal ui",
      "TUI",
      "git",
      "github",
      "lazygit alternative",
      "macos",
      "linux",
      "windows",
    ],
    siteName: "gitpad",
    ogLocale: "en_US",
  },
  nav: {
    about: "About",
    features: "Features",
    demo: "Demo",
    install: "Install",
    keys: "Keys",
    github: "GitHub",
  },
  hero: {
    eyebrow: "Open source · Go · single binary",
    title: "Your whole Git history,\non one screen.",
    subtitle:
      "gitpad is a terminal UI for Git: a branch tree, a colored commit graph, changed files and commit details side by side — with context menus for checkout, merge, rebase, cherry-pick and reset, a commit workspace and a push dialog.",
    tryDemo: "Try the demo",
    install: "Install",
    copy: "Copy",
    copied: "Copied!",
  },
  demo: {
    title: "Try it right here",
    subtitle:
      "This is a faithful re-creation of gitpad running in your browser on a fictional repository. Click it, then use the keys — it behaves like the real thing.",
    focusTitle: "Click to start",
    focusHint: "then drive it with your keyboard — arrows, enter, /, c, P…",
    focusOn: "keyboard active",
    reset: "Reset demo",
    keys: [
      ["↑ ↓ / j k", "move"],
      ["1 2 3 / tab", "switch pane"],
      ["enter", "actions / open"],
      ["/", "search"],
      ["c", "commit"],
      ["P", "push"],
      ["esc", "back"],
    ],
  },
  about: {
    title: "A Git TUI built around the log",
    paragraphs: [
      "gitpad is a terminal user interface (TUI) for Git. Instead of stitching together git log, git branch and git diff, you get the commit graph, the branch tree, the changed files and the commit details on one screen — the way a desktop Git client shows them, inside the terminal you already work in.",
      "It is a single Go binary that drives your own git, so hooks, credential helpers and signing keys keep working. Repositories hosted on GitHub get check status next to every commit, and a commit workspace and push dialog cover the everyday GitHub workflow without leaving the keyboard.",
    ],
    facts: [
      ["Platforms", "macOS · Linux · Windows"],
      ["Install", "Homebrew · apt · Scoop · go install"],
      ["License", "MIT, open source"],
      ["Language", "Go (bubbletea)"],
    ],
  },
  features: {
    title: "Everything the log view should have",
    subtitle: "Designed to look and feel like a desktop Git client — without leaving the terminal.",
    items: [
      {
        title: "Commit graph",
        body: "Lanes, merges and branch heads drawn per commit, with ref chips for HEAD, branches, remotes and tags.",
      },
      {
        title: "Context menus",
        body: "Right-click or press Enter on any commit or branch: checkout, merge, rebase, cherry-pick, revert, reset, new branch or tag.",
      },
      {
        title: "Commit workspace",
        body: "Check the files you want, write a multi-line message with history recall, preview the diff of each file, commit or commit & push.",
      },
      {
        title: "Push dialog",
        body: "See exactly which commits will be pushed, get warned when the branch is behind, toggle force-with-lease or tags.",
      },
      {
        title: "CI status",
        body: "✓ ✗ ◌ next to every commit from GitHub checks, with the individual runs and durations in the details pane.",
      },
      {
        title: "Search & filters",
        body: "One search box for messages, authors and hashes, a branch picker with type-to-filter, and a file-history filter.",
      },
    ],
  },
  install: {
    title: "Install in one line",
    subtitle:
      "Pre-built binaries for macOS, Linux and Windows. gitpad shells out to your own git, so hooks, credential helpers and signing keep working.",
    tabs: { mac: "macOS", debian: "Debian / Ubuntu", windows: "Windows", go: "Go" },
    note: "Then run gitpad inside any repository.",
    releases: "All releases",
  },
  keys: {
    title: "Keyboard first",
    subtitle: "Every action is a keystroke; the mouse works too.",
    groups: [
      {
        title: "Navigation",
        rows: [
          ["tab · 1 2 3", "switch pane"],
          ["j k ↑ ↓", "move"],
          ["g / G", "top / bottom"],
          ["← →", "fold / unfold, then previous / next pane"],
          ["↑ at top", "jump to the search bar"],
        ],
      },
      {
        title: "Log",
        rows: [
          ["enter / m / right-click", "commit actions"],
          ["/", "search message, author or hash"],
          ["A", "all branches ↔ current"],
          ["y", "copy hash"],
          ["v", "new version tag (patch / minor / major)"],
        ],
      },
      {
        title: "Commit & push",
        rows: [
          ["c / C", "commit workspace"],
          ["enter", "check / uncheck a file"],
          ["⌃S · ⌃P", "commit · commit & push"],
          ["P", "push dialog"],
          ["p", "pull (merge / rebase / fetch)"],
        ],
      },
      {
        title: "Diff",
        rows: [
          ["enter", "open diff"],
          ["↑ ↓", "next / previous change block"],
          ["n / p", "next / previous file"],
          ["esc", "back"],
        ],
      },
    ],
  },
  footer: {
    license: "MIT licensed",
    source: "Source on GitHub",
    lang: "언어: 한국어",
  },
} as const;

/** Same shape as `en`, with string literals widened so other locales fit. */
type Widen<T> = T extends string
  ? string
  : T extends readonly (infer U)[]
    ? readonly Widen<U>[]
    : T extends object
      ? { readonly [K in keyof T]: Widen<T[K]> }
      : T;

export type Dict = Widen<typeof en>;
