import { useState } from "react";
import { useT } from "../i18n";
import { Section } from "./Section";
import { CopyButton } from "./CopyButton";

type Tab = "mac" | "debian" | "windows" | "go";

const commands: Record<Tab, string[]> = {
  mac: [
    "brew tap jinhyo-dev/gitpad https://github.com/jinhyo-dev/gitpad",
    "brew install --cask gitpad",
  ],
  debian: [
    "curl -fsSL https://jinhyo-dev.github.io/gitpad/apt/key.gpg | sudo gpg --dearmor -o /usr/share/keyrings/gitpad.gpg",
    'echo "deb [signed-by=/usr/share/keyrings/gitpad.gpg] https://jinhyo-dev.github.io/gitpad/apt stable main" | sudo tee /etc/apt/sources.list.d/gitpad.list',
    "sudo apt-get update && sudo apt-get install gitpad",
  ],
  windows: ["scoop bucket add gitpad https://github.com/jinhyo-dev/gitpad", "scoop install gitpad"],
  go: ["go install github.com/jinhyo-dev/gitpad@latest"],
};

export function Install() {
  const t = useT();
  const [tab, setTab] = useState<Tab>("mac");
  const tabs = Object.keys(commands) as Tab[];
  const text = commands[tab].join("\n");
  return (
    <Section id="install" title={t.install.title} subtitle={t.install.subtitle}>
      <div className="overflow-hidden rounded-xl border border-surface2 bg-surface">
        <div className="flex items-center gap-1 border-b border-surface2 bg-mantle px-2 pt-2">
          {tabs.map((k) => (
            <button
              key={k}
              type="button"
              onClick={() => setTab(k)}
              className={
                "rounded-t-md px-3 py-1.5 text-sm font-medium " +
                (tab === k ? "bg-surface text-text" : "text-muted hover:text-text")
              }
            >
              {t.install.tabs[k]}
            </button>
          ))}
          <div className="ml-auto pb-2">
            <CopyButton text={text} />
          </div>
        </div>
        <pre className="overflow-x-auto px-5 py-4 font-mono text-sm leading-7 text-text">
          {commands[tab].map((c) => (
            <div key={c}>
              <span className="select-none text-dim">$ </span>
              {c}
            </div>
          ))}
        </pre>
      </div>
      <p className="mt-4 text-sm text-muted">
        {t.install.note}{" "}
        <a
          href="https://github.com/jinhyo-dev/gitpad/releases"
          className="text-accent hover:underline"
          target="_blank"
          rel="noreferrer"
        >
          {t.install.releases} ↗
        </a>
      </p>
    </Section>
  );
}
