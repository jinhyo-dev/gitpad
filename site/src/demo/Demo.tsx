import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  type KeyboardEvent,
  type ReactNode,
} from "react";
import { useT } from "../i18n";
import { Section } from "../components/Section";
import { Logo } from "../components/Logo";
import { buildGraph } from "./graph";
import { makeRepo, type Commit, type FileChange, type Repo } from "./data";

// ---------------------------------------------------------------------------
// A faithful, self-contained re-creation of gitpad's UI for the browser.
// ---------------------------------------------------------------------------

type Pane = "branches" | "log" | "changes";
type MenuItem = { label: string; key?: string; danger?: boolean; sep?: boolean; run?: () => void };
type Menu = { title: string; items: MenuItem[]; cur: number };
type Toast = { text: string; kind: "ok" | "err" | "info" };
type CommitWS = {
  focus: "files" | "message" | "buttons" | "diff";
  checked: Set<string>;
  message: string;
  button: 0 | 1;
  cur: number;
};

type State = {
  repo: Repo;
  ahead: number;
  focus: Pane;
  lcur: number;
  bcur: number;
  fcur: number;
  search: { active: boolean; text: string; applied: string };
  branchFilter: string | null;
  diff: { open: boolean; scroll: number } | null;
  menu: Menu | null;
  commit: CommitWS | null;
  push: boolean;
  quit: boolean;
  help: boolean;
  toast: Toast | null;
};

const LANES = [
  "#89b4fa",
  "#cba6f7",
  "#94e2d5",
  "#a6e3a1",
  "#f9e2af",
  "#fab387",
  "#f5c2e7",
  "#89dceb",
];
const ROWS = 22; // visible rows inside the panes

function initial(): State {
  return {
    repo: makeRepo(),
    ahead: 2,
    focus: "log",
    lcur: 0,
    bcur: 0,
    fcur: 0,
    search: { active: false, text: "", applied: "" },
    branchFilter: null,
    diff: null,
    menu: null,
    commit: null,
    push: false,
    quit: false,
    help: false,
    toast: null,
  };
}

function short(h: string) {
  return h.slice(0, 8);
}

function fmtDate(d: Date) {
  return `${d.getMonth() + 1}/${String(d.getDate()).padStart(2, "0")} ${String(d.getHours()).padStart(2, "0")}:${String(d.getMinutes()).padStart(2, "0")}`;
}

/** Commits reachable from a hash (used by the branch filter). */
function reachable(commits: Commit[], from: string): Set<string> {
  const byHash = new Map(commits.map((c) => [c.hash, c]));
  const seen = new Set<string>();
  const stack = [from];
  while (stack.length) {
    const h = stack.pop()!;
    if (seen.has(h)) continue;
    seen.add(h);
    for (const p of byHash.get(h)?.parents ?? []) stack.push(p);
  }
  return seen;
}

export function Demo() {
  const t = useT();
  const [s, setS] = useState<State>(initial);
  const [focused, setFocused] = useState(false);
  const boxRef = useRef<HTMLDivElement>(null);
  const toastTimer = useRef<number | undefined>(undefined);

  const toast = useCallback((text: string, kind: Toast["kind"] = "ok") => {
    setS((p) => ({ ...p, toast: { text, kind } }));
    window.clearTimeout(toastTimer.current);
    toastTimer.current = window.setTimeout(() => setS((p) => ({ ...p, toast: null })), 2500);
  }, []);

  // Visible commits after search / branch filter.
  const commits = useMemo(() => {
    let list = s.repo.commits;
    if (s.branchFilter) {
      const b = s.repo.branches.find((x) => x.name === s.branchFilter);
      if (b) {
        const ok = reachable(list, b.hash);
        list = list.filter((c) => ok.has(c.hash));
      }
    }
    const q = s.search.applied.toLowerCase();
    if (q)
      list = list.filter(
        (c) =>
          c.subject.toLowerCase().includes(q) ||
          c.author.name.toLowerCase().includes(q) ||
          c.hash.startsWith(q),
      );
    return list;
  }, [s.repo, s.branchFilter, s.search.applied]);
  const hasLocal = s.repo.local.length > 0 && !s.search.applied && !s.branchFilter;
  const rows = useMemo(
    () =>
      buildGraph(
        commits.map((c) => ({ hash: c.hash, parents: c.parents })),
        LANES.length,
      ),
    [commits],
  );
  const graphW = Math.min(16, Math.max(1, ...rows.map((r) => r.cells.length)));
  const logLen = commits.length + (hasLocal ? 1 : 0);
  const commitAt = (row: number): Commit | null => commits[hasLocal ? row - 1 : row] ?? null;
  const selected = commitAt(s.lcur);
  const isLocalRow = hasLocal && s.lcur === 0;
  const files: FileChange[] = isLocalRow ? s.repo.local : (selected?.files ?? []);
  const fileNode = files[s.fcur] ?? null;

  // Branch tree rows.
  const branchRows = useMemo(() => {
    const locals = s.repo.branches.filter((b) => b.kind === "local");
    const remotes = s.repo.branches.filter((b) => b.kind === "remote");
    const tags = s.repo.branches.filter((b) => b.kind === "tag");
    const out: {
      label: string;
      kind: "head" | "section" | "leaf";
      name?: string;
      depth: number;
      extra?: string;
    }[] = [];
    out.push({ label: "main", kind: "head", depth: 0, extra: s.ahead ? `↑${s.ahead}` : "" });
    out.push({ label: "Local", kind: "section", depth: 0, extra: String(locals.length) });
    for (const b of locals)
      out.push({
        label: b.name,
        kind: "leaf",
        name: b.name,
        depth: 1,
        extra: b.name === "main" && s.ahead ? `↑${s.ahead}` : "",
      });
    out.push({ label: "Remote", kind: "section", depth: 0, extra: String(remotes.length) });
    out.push({ label: "origin", kind: "section", depth: 1, extra: String(remotes.length) });
    for (const b of remotes)
      out.push({ label: b.name.replace(/^origin\//, ""), kind: "leaf", name: b.name, depth: 2 });
    out.push({ label: "Tags", kind: "section", depth: 0, extra: String(tags.length) });
    for (const b of tags) out.push({ label: b.name, kind: "leaf", name: b.name, depth: 1 });
    return out;
  }, [s.repo.branches, s.ahead]);

  // ----- actions ------------------------------------------------------------
  const closeOverlays = (p: State): State => ({
    ...p,
    menu: null,
    push: false,
    quit: false,
    help: false,
  });

  const commitMenu = (c: Commit): Menu => ({
    title: `${short(c.hash)}  ${c.subject}`,
    cur: 0,
    items: [
      {
        label: "Checkout revision (detached)",
        key: "c",
        run: () => toast(`Checkout ${short(c.hash)}`),
      },
      {
        label: "New branch here…",
        key: "b",
        run: () => toast(`Branch created at ${short(c.hash)}`),
      },
      { label: "New tag here…", key: "t", run: () => toast(`Tag created at ${short(c.hash)}`) },
      { sep: true, label: "" },
      { label: "Cherry-pick", key: "p", run: () => toast(`Cherry-pick ${short(c.hash)}`) },
      { label: "Revert commit", key: "v", run: () => toast(`Revert ${short(c.hash)}`) },
      {
        label: "Reset main to here ▸",
        key: "r",
        run: () => toast(`Reset --soft to ${short(c.hash)}`),
      },
      { sep: true, label: "" },
      { label: "Copy revision number", key: "y", run: () => toast(`copied hash ${short(c.hash)}`) },
      {
        label: "Open checks in browser",
        key: "k",
        run: () => toast("Opening GitHub checks…", "info"),
      },
    ],
  });
  const localMenu = (): Menu => ({
    title: "Uncommitted changes",
    cur: 0,
    items: [
      { label: "Commit…", key: "c", run: () => openCommit() },
      { label: "Push…", key: "P", run: () => setS((p) => ({ ...closeOverlays(p), push: true })) },
      { sep: true, label: "" },
      { label: "Stash changes", key: "S", run: () => toast("Stash") },
      {
        label: "Discard all changes",
        key: "D",
        danger: true,
        run: () => toast("Discard all (not in the demo)", "err"),
      },
    ],
  });
  const branchMenu = (name: string): Menu => ({
    title: name,
    cur: 0,
    items: [
      {
        label: "Show in log",
        key: "s",
        run: () =>
          setS((p) => ({ ...closeOverlays(p), branchFilter: name, lcur: 0, focus: "log" })),
      },
      { label: "Checkout", key: "c", run: () => toast(`Checkout ${name}`) },
      { sep: true, label: "" },
      { label: "Merge into main", key: "m", run: () => toast(`Merge ${name}`) },
      { label: "Rebase main onto " + name, key: "r", run: () => toast(`Rebase onto ${name}`) },
      { sep: true, label: "" },
      {
        label: "Delete",
        key: "d",
        danger: true,
        run: () => toast(`Delete ${name} (not in the demo)`, "err"),
      },
    ],
  });

  const openCommit = () =>
    setS((p) => {
      if (p.repo.local.length === 0) return p;
      const checked = new Set(p.repo.local.filter((f) => f.status !== "?").map((f) => f.path));
      return {
        ...closeOverlays(p),
        lcur: 0,
        focus: "log",
        diff: null,
        commit: p.commit
          ? { ...p.commit, focus: "files" }
          : { focus: "files", checked, message: "", button: 0, cur: 0 },
      };
    });

  const doCommit = (push: boolean) =>
    setS((p) => {
      const ws = p.commit;
      if (!ws) return p;
      const chosen = p.repo.local.filter((f) => ws.checked.has(f.path));
      if (chosen.length === 0) {
        toast("Select at least one file to commit", "err");
        return p;
      }
      const msg = ws.message.trim();
      if (!msg) {
        toast("Enter a commit message", "err");
        return { ...p, commit: { ...ws, focus: "message" } };
      }
      const [subject, ...rest] = msg.split("\n");
      const head = p.repo.commits[0];
      const newCommit: Commit = {
        hash: Array.from(
          { length: 40 },
          () => "0123456789abcdef"[Math.floor(Math.random() * 16)],
        ).join(""),
        parents: [head.hash],
        author: { name: "You", email: "you@example.com" },
        when: new Date(),
        subject,
        body: rest.join("\n").trim() || undefined,
        refs: head.refs.filter((r) => r.kind === "head" || r.name === "main"),
        files: chosen.map((f) => ({ ...f, status: f.status === "?" ? "A" : f.status })),
        ci: "pending",
      };
      const commits = [
        newCommit,
        { ...head, refs: head.refs.filter((r) => r.kind !== "head" && r.name !== "main") },
        ...p.repo.commits.slice(1),
      ];
      const branches = p.repo.branches.map((b) =>
        b.name === "main" ? { ...b, hash: newCommit.hash } : b,
      );
      const local = p.repo.local.filter((f) => !ws.checked.has(f.path));
      toast(`Commit ${short(newCommit.hash)} · ${subject}`);
      return {
        ...p,
        repo: { ...p.repo, commits, branches, head: newCommit.hash, local },
        ahead: p.ahead + 1,
        commit: null,
        lcur: hasLocalAfter(local) ? 1 : 0,
        push,
        diff: null,
      };
    });
  const hasLocalAfter = (local: FileChange[]) => local.length > 0;

  const doPush = () =>
    setS((p) => {
      if (p.ahead === 0) {
        toast("Nothing to push — main is up to date", "info");
        return { ...p, push: false };
      }
      const headHash = p.repo.commits[0].hash;
      const commits = p.repo.commits.map((c) => {
        const refs = c.refs.filter((r) => r.name !== "origin/main");
        return c.hash === headHash
          ? { ...c, refs: [...refs, { name: "origin/main", kind: "remote" as const }] }
          : { ...c, refs };
      });
      const branches = p.repo.branches.map((b) =>
        b.name === "origin/main"
          ? { ...b, hash: headHash }
          : b.name === "main"
            ? { ...b, ahead: 0 }
            : b,
      );
      toast(`Push main → origin/main (${p.ahead} commits)`);
      return { ...p, repo: { ...p.repo, commits, branches }, ahead: 0, push: false };
    });

  const reset = () => setS(initial());

  // ----- keyboard -----------------------------------------------------------
  const onKey = (e: KeyboardEvent<HTMLDivElement>) => {
    const k = e.key;
    const ctrl = e.ctrlKey || e.metaKey;
    const handled = () => e.preventDefault();

    // Overlays first.
    if (s.quit) {
      if (k === "q" || k === "Enter" || k === "y")
        toast("Thanks for trying gitpad — install it to keep going!", "info");
      setS((p) => ({ ...p, quit: false }));
      return handled();
    }
    if (s.help) {
      setS((p) => ({ ...p, help: false }));
      return handled();
    }
    if (s.push) {
      if (k === "Enter") doPush();
      else if (k === "Escape" || k === "q") setS((p) => ({ ...p, push: false }));
      return handled();
    }
    if (s.menu) {
      const m = s.menu;
      const move = (d: number) => {
        let i = m.cur;
        for (let n = 0; n < m.items.length; n++) {
          i = (i + d + m.items.length) % m.items.length;
          if (!m.items[i].sep) break;
        }
        setS((p) => ({ ...p, menu: { ...m, cur: i } }));
      };
      if (k === "ArrowDown" || k === "j") move(1);
      else if (k === "ArrowUp" || k === "k") move(-1);
      else if (k === "Enter") {
        setS((p) => ({ ...p, menu: null }));
        m.items[m.cur].run?.();
      } else if (k === "Escape") setS((p) => ({ ...p, menu: null }));
      else {
        const hit = m.items.find((it) => it.key === k);
        if (hit) {
          setS((p) => ({ ...p, menu: null }));
          hit.run?.();
        }
      }
      return handled();
    }
    if (s.search.active) {
      if (k === "Enter" || k === "ArrowDown")
        setS((p) => ({
          ...p,
          search: { ...p.search, active: false, applied: p.search.text },
          lcur: 0,
        }));
      else if (k === "Escape")
        setS((p) => ({ ...p, search: { ...p.search, active: false, text: p.search.applied } }));
      else if (k === "Backspace")
        setS((p) => ({ ...p, search: { ...p.search, text: p.search.text.slice(0, -1) } }));
      else if (k.length === 1 && !ctrl)
        setS((p) => ({ ...p, search: { ...p.search, text: p.search.text + k } }));
      return handled();
    }
    if (s.commit) {
      const ws = s.commit;
      if (ctrl && (k === "s" || k === "S")) {
        doCommit(false);
        return handled();
      }
      if (ctrl && (k === "p" || k === "P")) {
        doCommit(true);
        return handled();
      }
      const order: CommitWS["focus"][] = ["files", "message", "buttons", "diff"];
      if (k === "Tab") {
        const i = order.indexOf(ws.focus);
        setS((p) => ({ ...p, commit: { ...ws, focus: order[(i + (e.shiftKey ? 3 : 1)) % 4] } }));
        return handled();
      }
      if (ws.focus === "message") {
        if (k === "Escape") setS((p) => ({ ...p, commit: { ...ws, focus: "files" } }));
        return; // the textarea handles typing itself
      }
      if (k === "1" || k === "2" || k === "3") {
        setS((p) => ({
          ...p,
          commit: { ...ws, focus: k === "1" ? "files" : k === "2" ? "message" : "diff" },
        }));
        return handled();
      }
      if (ws.focus === "files") {
        const n = s.repo.local.length;
        if (k === "ArrowDown" || k === "j")
          setS((p) => ({ ...p, commit: { ...ws, cur: Math.min(ws.cur + 1, n - 1) } }));
        else if (k === "ArrowUp" || k === "k")
          setS((p) => ({ ...p, commit: { ...ws, cur: Math.max(ws.cur - 1, 0) } }));
        else if (k === "Enter" || k === " ") {
          const path = s.repo.local[ws.cur]?.path;
          if (path) {
            const checked = new Set(ws.checked);
            if (checked.has(path)) checked.delete(path);
            else checked.add(path);
            setS((p) => ({ ...p, commit: { ...ws, checked } }));
          }
        } else if (k === "a") {
          const all =
            ws.checked.size === n ? new Set<string>() : new Set(s.repo.local.map((f) => f.path));
          setS((p) => ({ ...p, commit: { ...ws, checked: all } }));
        } else if (k === "ArrowRight" || k === "l")
          setS((p) => ({ ...p, commit: { ...ws, focus: "diff" } }));
        else if (k === "Escape" || k === "q") setS((p) => ({ ...p, commit: null }));
        return handled();
      }
      if (ws.focus === "buttons") {
        if (k === "ArrowLeft" || k === "ArrowRight" || k === "h" || k === "l")
          setS((p) => ({ ...p, commit: { ...ws, button: ws.button === 0 ? 1 : 0 } }));
        else if (k === "Enter" || k === " ") doCommit(ws.button === 1);
        else if (k === "ArrowUp" || k === "k")
          setS((p) => ({ ...p, commit: { ...ws, focus: "message" } }));
        else if (k === "Escape") setS((p) => ({ ...p, commit: null }));
        return handled();
      }
      if (ws.focus === "diff") {
        if (k === "Escape" || k === "ArrowLeft" || k === "h")
          setS((p) => ({ ...p, commit: { ...ws, focus: "files" } }));
        else if (k === "n" || k === "ArrowDown" || k === "j")
          setS((p) => ({
            ...p,
            commit: { ...ws, cur: Math.min(ws.cur + 1, s.repo.local.length - 1) },
          }));
        else if (k === "p" || k === "ArrowUp" || k === "k")
          setS((p) => ({ ...p, commit: { ...ws, cur: Math.max(ws.cur - 1, 0) } }));
        return handled();
      }
      return;
    }

    // Global keys.
    if (k === "?") return (setS((p) => ({ ...p, help: true })), handled());
    if (k === "/")
      return (setS((p) => ({ ...p, search: { ...p.search, active: true } })), handled());
    if (k === "Tab") {
      const order: Pane[] = ["branches", "log", "changes"];
      const i = order.indexOf(s.focus);
      setS((p) => ({ ...p, focus: order[(i + (e.shiftKey ? 2 : 1)) % 3] }));
      return handled();
    }
    if (k === "1" || k === "2" || k === "3")
      return (
        setS((p) => ({ ...p, focus: (["branches", "log", "changes"] as Pane[])[Number(k) - 1] })),
        handled()
      );
    if (k === "c" || k === "C") return (openCommit(), handled());
    if (k === "P") return (setS((p) => ({ ...p, push: true })), handled());
    if (k === "p") return (toast("Pull (merge) — up to date"), handled());
    if (k === "f") return (toast("Fetch — up to date"), handled());
    if (k === "q") {
      if (s.diff) setS((p) => ({ ...p, diff: null }));
      else setS((p) => ({ ...p, quit: true }));
      return handled();
    }
    if (k === "Escape") {
      if (s.diff) setS((p) => ({ ...p, diff: null }));
      else if (s.search.applied || s.branchFilter)
        setS((p) => ({
          ...p,
          search: { active: false, text: "", applied: "" },
          branchFilter: null,
          lcur: 0,
        }));
      return handled();
    }
    if (k === "A")
      return (
        setS((p) => ({ ...p, branchFilter: p.branchFilter ? null : "main", lcur: 0 })), handled()
      );

    if (s.diff && s.focus !== "branches") {
      const d = s.diff;
      if (k === "ArrowDown" || k === "j")
        setS((p) => ({ ...p, diff: { ...d, scroll: d.scroll + 3 } }));
      else if (k === "ArrowUp" || k === "k")
        setS((p) => ({ ...p, diff: { ...d, scroll: Math.max(0, d.scroll - 3) } }));
      else if (k === "n")
        setS((p) => ({
          ...p,
          fcur: Math.min(p.fcur + 1, files.length - 1),
          diff: { open: true, scroll: 0 },
        }));
      else if (k === "p")
        setS((p) => ({ ...p, fcur: Math.max(p.fcur - 1, 0), diff: { open: true, scroll: 0 } }));
      return handled();
    }

    const nav = (cur: number, n: number): number | null => {
      if (k === "ArrowDown" || k === "j") return Math.min(cur + 1, n - 1);
      if (k === "ArrowUp" || k === "k") return Math.max(cur - 1, 0);
      if (k === "g") return 0;
      if (k === "G") return n - 1;
      return null;
    };
    if (s.focus === "log") {
      const nx = nav(s.lcur, logLen);
      if (nx !== null) {
        if (nx === s.lcur && (k === "ArrowUp" || k === "k"))
          setS((p) => ({ ...p, search: { ...p.search, active: true } }));
        else setS((p) => ({ ...p, lcur: nx, fcur: 0 }));
        return handled();
      }
      if (k === "ArrowLeft") return (setS((p) => ({ ...p, focus: "branches" })), handled());
      if (k === "ArrowRight") return (setS((p) => ({ ...p, focus: "changes" })), handled());
      if (k === "Enter" || k === "m") {
        setS((p) => ({
          ...p,
          menu: isLocalRow ? localMenu() : selected ? commitMenu(selected) : null,
        }));
        return handled();
      }
      if (k === "y" && selected) return (toast(`copied hash ${short(selected.hash)}`), handled());
      if (k === "v")
        return (toast("New version tag: v1.5.0 (patch / minor / major)", "info"), handled());
      return;
    }
    if (s.focus === "branches") {
      const nx = nav(s.bcur, branchRows.length);
      if (nx !== null) return (setS((p) => ({ ...p, bcur: nx })), handled());
      if (k === "ArrowRight") return (setS((p) => ({ ...p, focus: "log" })), handled());
      const row = branchRows[s.bcur];
      if ((k === "Enter" || k === "m") && row.kind === "leaf" && row.name)
        return (setS((p) => ({ ...p, menu: branchMenu(row.name!) })), handled());
      if (k === "s" && row.kind === "leaf" && row.name)
        return (setS((p) => ({ ...p, branchFilter: row.name!, lcur: 0, focus: "log" })), handled());
      return;
    }
    if (s.focus === "changes") {
      const nx = nav(s.fcur, files.length);
      if (nx !== null)
        return (
          setS((p) => ({ ...p, fcur: nx, diff: p.diff ? { open: true, scroll: 0 } : null })),
          handled()
        );
      if (k === "ArrowLeft") return (setS((p) => ({ ...p, focus: "log" })), handled());
      if (k === "ArrowRight") return (setS((p) => ({ ...p, focus: "branches" })), handled());
      if (k === "Enter" && fileNode)
        return (
          setS((p) => ({ ...p, diff: p.diff ? null : { open: true, scroll: 0 } })), handled()
        );
      if (k === "y" && fileNode) return (toast(`copied path ${fileNode.path}`), handled());
      if (k === "d" && isLocalRow) return (toast("Discard (not in the demo)", "err"), handled());
      return;
    }
  };

  // "Try the demo" (#demo) lands here: focus the box so keys work right away.
  useEffect(() => {
    const onHash = () => {
      if (window.location.hash === "#demo") {
        setFocused(true);
        boxRef.current?.focus({ preventScroll: true });
      }
    };
    onHash();
    window.addEventListener("hashchange", onHash);
    return () => window.removeEventListener("hashchange", onHash);
  }, []);

  // Keep the cursor row in view.
  const lscroll = Math.max(0, Math.min(s.lcur - Math.floor(ROWS / 2), logLen - ROWS));
  useEffect(() => {
    if (boxRef.current && focused && !s.commit?.focus)
      boxRef.current.focus({ preventScroll: true });
  }, [focused, s.commit?.focus]);

  // ----- render helpers -----------------------------------------------------
  const chip = (name: string, kind: string, i: number) => {
    const cls =
      kind === "head"
        ? "bg-accent text-base font-bold"
        : kind === "local"
          ? "bg-green/20 text-green"
          : kind === "remote"
            ? "bg-magenta/20 text-magenta"
            : "bg-yellow/20 text-yellow";
    return (
      <span key={i} className={"ml-1 rounded px-1 " + cls}>
        {kind === "tag" ? "⌂ " : ""}
        {name}
      </span>
    );
  };
  const ci = (c: Commit) =>
    c.ci === "ok" ? (
      <span className="text-green">✓</span>
    ) : c.ci === "fail" ? (
      <span className="text-red">✗</span>
    ) : (
      <span className="text-teal">◌</span>
    );
  const graph = (i: number) => (
    <span className="inline-block" style={{ width: `${graphW}ch` }}>
      {rows[i]?.cells.slice(0, graphW).map((cell, j) => (
        <span key={j} style={{ color: cell.color >= 0 ? LANES[cell.color] : undefined }}>
          {cell.ch === "●" && commits[i].hash === s.repo.head ? "◉" : cell.ch}
        </span>
      ))}
    </span>
  );

  const frame = (title: string, count: string, body: ReactNode, active: boolean, extra = "") => (
    <div
      className={
        "relative flex min-h-0 min-w-0 flex-col rounded-lg border " +
        (active ? "border-accent" : "border-overlay") +
        " " +
        extra
      }
    >
      <div
        className={
          "absolute -top-2.5 left-3 bg-surface px-1 text-xs font-bold " +
          (active ? "text-accent" : "text-muted")
        }
      >
        {title} {count && <span className="font-normal text-dim">{count}</span>}
      </div>
      <div className="min-h-0 flex-1 overflow-hidden px-2 pt-2">{body}</div>
    </div>
  );

  const rowCls = (sel: boolean, active: boolean) =>
    "tui-line rounded px-1 " + (sel ? (active ? "bg-selection" : "bg-surface2") : "");

  // Panes ---------------------------------------------------------------------
  const branchesPane = frame(
    "Branches",
    "",
    branchRows.slice(0, ROWS).map((r, i) => (
      <div
        key={i}
        className={rowCls(i === s.bcur, s.focus === "branches")}
        onClick={() => setS((p) => ({ ...p, focus: "branches", bcur: i }))}
      >
        <span style={{ paddingLeft: `${r.depth * 2}ch` }} />
        {r.kind === "head" && <span className="text-accent">◉ </span>}
        {r.kind === "section" && <span className="text-muted">▾ </span>}
        {r.kind === "leaf" && (r.name === "main" ? <span className="text-accent">● </span> : "  ")}
        <span
          className={
            r.kind === "section"
              ? "font-bold text-muted"
              : r.name === "main"
                ? "font-bold text-accent"
                : r.label.startsWith("v") && r.depth === 1
                  ? "text-yellow"
                  : ""
          }
        >
          {r.label}
        </span>
        {r.extra && <span className="float-right text-dim">{r.extra}</span>}
      </div>
    )),
    s.focus === "branches",
    "w-52 shrink-0",
  );

  const logRows: ReactNode[] = [];
  for (let i = lscroll; i < Math.min(logLen, lscroll + ROWS); i++) {
    const sel = i === s.lcur;
    if (hasLocal && i === 0) {
      logRows.push(
        <div
          key="local"
          className={rowCls(sel, s.focus === "log")}
          onClick={() => setS((p) => ({ ...p, focus: "log", lcur: 0 }))}
        >
          <span className="inline-block w-[27ch]" />
          <span className="text-yellow">◌ </span>
          <span className="italic text-yellow">Uncommitted changes</span>
          <span className="text-muted"> · {s.repo.local.length} files</span>
        </div>,
      );
      continue;
    }
    const c = commitAt(i)!;
    const ci_ = commits.indexOf(c);
    const isHead = c.hash === s.repo.head;
    logRows.push(
      <div
        key={c.hash}
        className={rowCls(sel, s.focus === "log") + " flex"}
        onClick={() => setS((p) => ({ ...p, focus: "log", lcur: i, fcur: 0 }))}
        onContextMenu={(e) => {
          e.preventDefault();
          setS((p) => ({ ...p, focus: "log", lcur: i, menu: commitMenu(c) }));
        }}
      >
        <span
          className={"w-[14ch] shrink-0 overflow-hidden " + (isHead ? "text-text" : "text-muted")}
        >
          {c.author.name}
        </span>
        <span className="w-[12ch] shrink-0 text-dim">{fmtDate(c.when)}</span>
        <span className="w-[2ch] shrink-0">{ci(c)}</span>
        <span className="shrink-0">{graph(ci_)} </span>
        <span
          className={
            "min-w-0 flex-1 truncate " +
            (isHead ? "font-bold" : c.parents.length > 1 ? "text-muted" : "")
          }
        >
          {c.subject}
        </span>
        {c.refs.length > 0 && (
          <span className="shrink-0">{c.refs.map((r, j) => chip(r.name, r.kind, j))}</span>
        )}
      </div>,
    );
  }

  const diffLines = (text: string, scroll: number) =>
    text
      .split("\n")
      .slice(scroll, scroll + ROWS)
      .map((l, i) => (
        <div
          key={i}
          className={
            "tui-line " +
            (l.startsWith("+") && !l.startsWith("+++")
              ? "bg-green/15 text-green"
              : l.startsWith("-") && !l.startsWith("---")
                ? "bg-red/15 text-red"
                : l.startsWith("@@")
                  ? "font-bold text-magenta"
                  : l.startsWith("diff") || l.startsWith("---") || l.startsWith("+++")
                    ? "text-dim"
                    : "")
          }
        >
          {l}
        </div>
      ));

  const filesPane = frame(
    isLocalRow ? "Local Changes" : "Changes",
    String(files.length),
    files.length === 0 ? (
      <div className="tui-line text-muted"> Select a commit</div>
    ) : (
      files.map((f, i) => (
        <div
          key={f.path}
          className={rowCls(i === s.fcur, s.focus === "changes")}
          onClick={() => setS((p) => ({ ...p, focus: "changes", fcur: i }))}
        >
          {isLocalRow && (
            <span className={f.status === "?" ? "text-dim" : "text-accent"}>
              {f.status === "?" ? "[ ] " : "[x] "}
            </span>
          )}
          <span
            className={
              "font-bold " +
              (f.status === "A"
                ? "text-green"
                : f.status === "?"
                  ? "text-teal"
                  : f.status === "D"
                    ? "text-red"
                    : "text-accent")
            }
          >
            {f.status}
          </span>{" "}
          <span
            className={
              f.status === "A" ? "text-green" : f.status === "?" ? "text-teal" : "text-accent"
            }
          >
            {f.path.split("/").pop()}
          </span>
          <span className="text-dim"> {f.path.split("/").slice(0, -1).join("/")}</span>
        </div>
      ))
    ),
    s.focus === "changes",
    "flex-1",
  );

  const detailsPane = frame(
    "Details",
    "",
    isLocalRow ? (
      <>
        <div className="tui-line font-bold">Uncommitted changes</div>
        <div className="tui-line text-muted">
          {s.repo.local.filter((f) => f.status !== "?").length} changed ·{" "}
          {s.repo.local.filter((f) => f.status === "?").length} untracked
        </div>
        <div className="tui-line text-dim">c commit… P push… enter diff</div>
      </>
    ) : selected ? (
      <>
        <div className="tui-line font-bold">{selected.subject}</div>
        {selected.body && <div className="tui-line text-muted">{selected.body}</div>}
        <div className="tui-line">
          <span className="text-accent">{selected.hash.slice(0, 12)}</span> {selected.author.name}{" "}
          <span className="text-dim">&lt;{selected.author.email}&gt;</span>
        </div>
        <div className="tui-line text-muted">
          {selected.when.toISOString().slice(0, 16).replace("T", " ")}
        </div>
        <div className="tui-line">{selected.refs.map((r, j) => chip(r.name, r.kind, j))}</div>
        <div className="tui-line">
          <span className="font-bold text-muted">Checks</span> {ci(selected)}{" "}
          <span className="text-muted">
            {selected.ci === "ok"
              ? "all checks passed"
              : selected.ci === "fail"
                ? "checks failed"
                : "checks running"}
          </span>
        </div>
        <div className="tui-line">
          {" "}
          {ci(selected)} test (ubuntu-latest) <span className="text-dim">1m 12s</span>
        </div>
        <div className="tui-line">
          {" "}
          {ci(selected)} build (windows) <span className="text-dim">2m 04s</span>
        </div>
      </>
    ) : (
      <div className="tui-line text-muted">Select a commit</div>
    ),
    false,
    "flex-1",
  );

  // Center pane variants.
  let center: ReactNode;
  let right: ReactNode = (
    <div className="flex w-[34%] shrink-0 flex-col gap-4">
      {filesPane}
      {detailsPane}
    </div>
  );
  if (s.commit) {
    const ws = s.commit;
    const cur = s.repo.local[ws.cur];
    center = frame(
      "Commit → main",
      `${s.repo.local.length} files`,
      <div className="flex h-full flex-col">
        <div className="tui-line text-muted">
          ▾ Changes{" "}
          <span className="float-right text-dim">
            {ws.checked.size}/{s.repo.local.length}
          </span>
        </div>
        {s.repo.local.map((f, i) => (
          <div
            key={f.path}
            className={rowCls(i === ws.cur, ws.focus === "files")}
            onClick={() => {
              const checked = new Set(ws.checked);
              if (checked.has(f.path)) checked.delete(f.path);
              else checked.add(f.path);
              setS((p) => ({ ...p, commit: { ...ws, cur: i, focus: "files", checked } }));
            }}
          >
            {"  "}
            <span className={ws.checked.has(f.path) ? "font-bold text-accent" : "text-dim"}>
              {ws.checked.has(f.path) ? "[x]" : "[ ]"}
            </span>{" "}
            <span className={"font-bold " + (f.status === "?" ? "text-teal" : "text-accent")}>
              {f.status}
            </span>{" "}
            <span className={f.status === "?" ? "text-teal" : "text-accent"}>
              {f.path.split("/").pop()}
            </span>
            <span className="text-dim"> {f.path.split("/").slice(0, -1).join("/")}</span>
          </div>
        ))}
        <div className="mt-auto border-t border-overlay pt-1">
          <div
            className={
              "tui-line text-xs font-bold " +
              (ws.focus === "message" ? "text-accent" : "text-muted")
            }
          >
            Commit message
          </div>
          <textarea
            value={ws.message}
            onChange={(e) => setS((p) => ({ ...p, commit: { ...ws, message: e.target.value } }))}
            onFocus={() => setS((p) => ({ ...p, commit: { ...ws, focus: "message" } }))}
            onKeyDown={(e) => {
              if ((e.ctrlKey || e.metaKey) && (e.key === "s" || e.key === "p")) {
                e.preventDefault();
                doCommit(e.key === "p");
              } else if (e.key === "Escape" || e.key === "Tab") {
                e.preventDefault();
                setS((p) => ({
                  ...p,
                  commit: { ...ws, focus: e.key === "Tab" ? "buttons" : "files" },
                }));
                boxRef.current?.focus({ preventScroll: true });
              }
            }}
            ref={(el) => {
              if (el && ws.focus === "message" && document.activeElement !== el)
                el.focus({ preventScroll: true });
            }}
            placeholder="Commit message — first line is the subject"
            className="h-20 w-full resize-none border-l-2 border-accent bg-transparent pl-2 font-mono text-[13px] leading-5 text-text outline-none placeholder:text-dim"
          />
          <div className="tui-line mt-1 flex items-center gap-2">
            <button
              type="button"
              onClick={() => doCommit(false)}
              className={
                "rounded px-3 " +
                (ws.focus === "buttons" && ws.button === 0
                  ? "bg-accent text-base font-bold"
                  : ws.focus !== "buttons"
                    ? "bg-accent text-base font-bold"
                    : "bg-surface2 text-muted")
              }
            >
              Commit
            </button>
            <button
              type="button"
              onClick={() => doCommit(true)}
              className={
                "rounded px-3 " +
                (ws.focus === "buttons" && ws.button === 1
                  ? "bg-accent text-base font-bold"
                  : "bg-surface2 text-muted")
              }
            >
              Commit &amp; Push
            </button>
            <span className="ml-auto text-dim">
              ⌃S commit · ⌃P &amp; push · {ws.checked.size}/{s.repo.local.length} files
            </span>
          </div>
        </div>
      </div>,
      ws.focus !== "diff",
      "flex-1",
    );
    right = (
      <div className="flex w-[46%] shrink-0 flex-col">
        {frame(
          `Diff · ${cur?.path.split("/").pop() ?? ""}`,
          "",
          cur ? diffLines(cur.diff, 0) : null,
          ws.focus === "diff",
          "flex-1",
        )}
      </div>
    );
  } else if (s.diff && fileNode) {
    center = frame(
      `Diff · ${fileNode.path}`,
      "esc to close",
      diffLines(fileNode.diff, s.diff.scroll),
      s.focus === "log",
      "flex-1",
    );
  } else {
    center = frame("Log", String(commits.length), logRows, s.focus === "log", "flex-1");
  }

  // Overlays ------------------------------------------------------------------
  const menuBox = s.menu && (
    <div className="absolute left-1/3 top-16 z-20 min-w-72 rounded-lg border border-accent bg-surface p-2 shadow-glow">
      <div className="tui-line font-bold text-muted">{s.menu.title}</div>
      <div className="mb-1 border-t border-overlay" />
      {s.menu.items.map((it, i) =>
        it.sep ? (
          <div key={i} className="my-1 border-t border-overlay" />
        ) : (
          <div
            key={i}
            className={
              "tui-line flex rounded px-1 " +
              (i === s.menu!.cur ? "bg-selection font-bold" : "") +
              (it.danger ? " text-red" : "")
            }
            onClick={() => {
              setS((p) => ({ ...p, menu: null }));
              it.run?.();
            }}
          >
            <span>{it.label}</span>
            <span className="ml-auto pl-6 text-dim">{it.key}</span>
          </div>
        ),
      )}
    </div>
  );
  const pushBox = s.push && (
    <div className="absolute left-1/2 top-1/2 z-20 w-[34rem] -translate-x-1/2 -translate-y-1/2 rounded-lg border border-accent bg-surface p-4 shadow-glow">
      <div className="tui-line font-bold">Push</div>
      <div className="tui-line mt-2">
        <span className="font-bold text-accent">main</span>
        <span className="text-muted"> → </span>
        <span className="text-magenta">origin/main</span>
      </div>
      <div className="tui-line mt-2 font-bold">
        {s.ahead ? `${s.ahead} commits will be pushed` : "Nothing to push — main is up to date"}
      </div>
      {s.repo.commits.slice(0, s.ahead).map((c) => (
        <div key={c.hash} className="tui-line">
          {" "}
          <span className="text-accent">{short(c.hash)}</span> {c.subject}
        </div>
      ))}
      <div className="tui-line mt-2">
        <span className="text-dim">[ ]</span> Force with lease (f){" "}
        <span className="text-dim">[ ]</span> Push tags (t)
      </div>
      <div className="tui-line mt-2 flex gap-2">
        <button
          type="button"
          onClick={doPush}
          className="rounded bg-accent px-3 font-bold text-base"
        >
          Push
        </button>
        <button
          type="button"
          onClick={() => setS((p) => ({ ...p, push: false }))}
          className="rounded bg-surface2 px-3 text-muted"
        >
          Cancel
        </button>
        <span className="ml-auto text-dim">enter push · esc cancel</span>
      </div>
    </div>
  );
  const quitBox = s.quit && (
    <div className="absolute left-1/2 top-1/2 z-20 w-96 -translate-x-1/2 -translate-y-1/2 rounded-lg border border-accent bg-surface p-4 shadow-glow">
      <div className="tui-line font-bold">Quit gitpad?</div>
      <div className="tui-line text-muted">Press q again or Enter to exit, Esc to stay.</div>
      <div className="tui-line mt-2">
        <span className="rounded bg-accent px-3 font-bold text-base">Quit</span>{" "}
        <span className="rounded bg-surface2 px-3 text-muted">Cancel</span>
      </div>
    </div>
  );
  const helpBox = s.help && (
    <div className="absolute left-1/2 top-1/2 z-20 -translate-x-1/2 -translate-y-1/2 rounded-lg border border-accent bg-surface p-4 shadow-glow">
      <div className="tui-line font-bold text-accent">
        gitpad <span className="font-normal text-muted">— keyboard reference</span>
      </div>
      {t.demo.keys.map(([k, d]) => (
        <div key={k} className="tui-line">
          <span className="inline-block w-[16ch] font-bold text-accent">{k}</span>
          <span className="text-muted">{d}</span>
        </div>
      ))}
      <div className="tui-line mt-2 text-dim">any key to close</div>
    </div>
  );
  const toastBox = s.toast && (
    <div
      className={
        "absolute left-1/2 top-12 z-30 -translate-x-1/2 rounded px-4 py-0.5 font-bold text-base " +
        (s.toast.kind === "ok" ? "bg-green" : s.toast.kind === "err" ? "bg-red" : "bg-accent")
      }
    >
      {s.toast.kind === "ok" ? "✓" : s.toast.kind === "err" ? "✗" : "ℹ"} {s.toast.text}
    </div>
  );

  const hints = s.commit
    ? "enter check  a check all  tab message  → diff  1 2 3 pane  ⌃S commit  esc back"
    : s.diff
      ? "↑↓ scroll  n/p next/prev file  esc back"
      : s.focus === "log"
        ? "enter actions  c commit  P push  p pull  f fetch  v version tag  / search  A all/head  y copy hash  ←→ section"
        : s.focus === "branches"
          ? "enter actions  s show in log  c checkout  ←→ section"
          : "enter diff  y copy path  ←→ section";

  return (
    <Section id="demo" title={t.demo.title} subtitle={t.demo.subtitle}>
      <div className="mb-3 flex flex-wrap items-center gap-3 text-sm text-muted">
        <span
          className={
            "rounded-full px-2.5 py-0.5 font-mono text-xs transition " +
            (focused ? "bg-green/20 text-green" : "bg-surface2 text-muted")
          }
        >
          {focused ? "● " + t.demo.focusOn : "○ " + t.demo.focusTitle}
        </span>
        <button
          type="button"
          onClick={reset}
          className="ml-auto rounded-md border border-surface2 px-2.5 py-1 text-xs hover:border-accent hover:text-text"
        >
          {t.demo.reset}
        </button>
      </div>
      <div
        ref={boxRef}
        tabIndex={0}
        onKeyDown={onKey}
        onFocus={() => setFocused(true)}
        onBlur={(e) => {
          if (!e.currentTarget.contains(e.relatedTarget as Node | null)) setFocused(false);
        }}
        onMouseDown={() => setFocused(true)}
        className={
          "tui relative select-none overflow-hidden rounded-xl border bg-surface text-text outline-none " +
          (focused ? "border-accent shadow-glow" : "border-surface2")
        }
        style={{ height: `${(ROWS + 6) * 20}px` }}
      >
        {/* header */}
        <div className="tui-line flex items-center gap-2 bg-mantle px-2">
          <Logo size={16} className="rounded" />
          <span className="rounded bg-accent px-1.5 font-bold text-base">gitpad</span>
          <span className="font-bold">acme-platform</span>
          <span className="text-dim">on</span>
          <span className="font-bold text-accent">main</span>
          <span className="text-dim">→ origin/main</span>
          {s.ahead > 0 && <span className="text-green">↑{s.ahead}</span>}
          <span className="ml-auto rounded bg-surface2 px-2 font-bold text-accent">Log</span>
          <span className="px-2 text-muted">Console</span>
          <span className="text-accent">?</span>
          <span className="text-muted">help</span>
        </div>
        {/* filter bar */}
        <div className="tui-line flex items-center gap-3 bg-mantle px-2">
          <span
            className={
              "rounded px-2 " +
              (s.search.active
                ? "bg-surface2 text-text"
                : s.search.applied
                  ? "bg-accent font-bold text-base"
                  : "bg-surface2 text-muted")
            }
          >
            ⌕{" "}
            {s.search.active
              ? s.search.text + "▏"
              : s.search.applied || "Search message, author or hash"}
          </span>
          <span
            className={
              "rounded px-2 " +
              (s.branchFilter ? "bg-accent font-bold text-base" : "bg-surface2 text-muted")
            }
          >
            Branch: {s.branchFilter ?? "All"} ▾
          </span>
          {(s.search.applied || s.branchFilter) && (
            <span className="ml-auto text-muted">
              <span className="font-bold text-accent">esc</span> clear filters
            </span>
          )}
        </div>
        {/* panes */}
        <div className="flex gap-4 px-3 pb-2 pt-4" style={{ height: `${(ROWS + 3) * 20}px` }}>
          {!s.commit && branchesPane}
          {s.commit ? <div className="w-52 shrink-0">{branchesPane}</div> : null}
          {center}
          {right}
        </div>
        {/* status bar */}
        <div className="tui-line flex bg-mantle px-2 text-muted">
          <span className="truncate">
            {hints.split("  ").map((h, i) => {
              const [k, ...d] = h.split(" ");
              return (
                <span key={i} className="mr-3">
                  <span className="font-bold text-accent">{k}</span> {d.join(" ")}
                </span>
              );
            })}
            <span className="font-bold text-red">q</span> exit
          </span>
          <span className="ml-auto text-dim">
            {Math.min(s.lcur + 1, logLen)}/{logLen}
          </span>
        </div>
        {menuBox}
        {pushBox}
        {quitBox}
        {helpBox}
        {toastBox}
        {!focused && (
          <div className="absolute inset-0 z-40 grid cursor-pointer place-items-center bg-base/55 backdrop-blur-[1.5px]">
            <div className="flex flex-col items-center gap-2 rounded-xl border border-accent/60 bg-surface/95 px-8 py-6 text-center shadow-glow">
              <div className="flex h-11 w-11 items-center justify-center rounded-full bg-accent text-base">
                <svg
                  width="22"
                  height="22"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="2"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  aria-hidden="true"
                >
                  <rect x="2" y="6" width="20" height="12" rx="2" />
                  <path d="M6 10h.01M10 10h.01M14 10h.01M18 10h.01M8 14h8" />
                </svg>
              </div>
              <div className="font-sans text-lg font-semibold text-text">{t.demo.focusTitle}</div>
              <div className="font-sans text-sm text-muted">{t.demo.focusHint}</div>
            </div>
          </div>
        )}
      </div>
      <div className="mt-3 flex flex-wrap gap-x-6 gap-y-1 text-sm text-muted">
        {t.demo.keys.map(([k, d]) => (
          <span key={k}>
            <span className="kbd">{k}</span> {d}
          </span>
        ))}
      </div>
    </Section>
  );
}
