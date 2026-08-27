import { useT } from "../i18n";
import { CopyButton } from "./CopyButton";

const BREW =
  "brew tap jinhyo-dev/gitpad https://github.com/jinhyo-dev/gitpad && brew install --cask gitpad";

export function Hero() {
  const t = useT();
  const [line1, line2] = t.hero.title.split("\n");
  return (
    <section className="mx-auto max-w-6xl px-5 pb-10 pt-20 sm:px-8 sm:pt-28">
      <p className="font-mono text-xs uppercase tracking-[0.2em] text-accent">{t.hero.eyebrow}</p>
      <h1 className="mt-4 text-5xl font-bold leading-[1.05] tracking-tight text-text sm:text-6xl">
        {line1}
        <br />
        <span className="bg-gradient-to-r from-accent via-magenta to-pink bg-clip-text text-transparent">
          {line2}
        </span>
      </h1>
      <p className="mt-6 max-w-2xl text-lg leading-relaxed text-muted">{t.hero.subtitle}</p>
      <div className="mt-8 flex flex-wrap items-center gap-3">
        <a
          href="#demo"
          className="rounded-lg bg-accent px-5 py-2.5 font-semibold text-base shadow-glow hover:brightness-110"
        >
          {t.hero.tryDemo}
        </a>
        <a
          href="#install"
          className="rounded-lg border border-surface2 px-5 py-2.5 font-semibold text-text hover:border-accent"
        >
          {t.hero.install}
        </a>
      </div>
      <div className="mt-6 flex max-w-4xl items-center gap-3 rounded-lg border border-surface2 bg-surface px-4 py-3 font-mono text-[13px] text-text">
        <span className="select-none text-dim">$</span>
        <code className="min-w-0 flex-1 overflow-x-auto whitespace-nowrap">{BREW}</code>
        <CopyButton text={BREW} />
      </div>
    </section>
  );
}
