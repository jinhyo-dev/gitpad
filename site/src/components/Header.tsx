import { pathFor, useLocale, useT } from "../i18n";
import { Logo } from "./Logo";

const REPO = "https://github.com/jinhyo-dev/gitpad";

export function Header() {
  const t = useT();
  const locale = useLocale();
  const other = locale === "en" ? "ko" : "en";
  const links: [string, string][] = [
    ["#about", t.nav.about],
    ["#features", t.nav.features],
    ["#gallery", t.nav.gallery],
    ["#demo", t.nav.demo],
    ["#install", t.nav.install],
    ["#keys", t.nav.keys],
  ];
  return (
    <header className="sticky top-0 z-40 border-b border-surface2/60 bg-base/70 backdrop-blur">
      <div className="mx-auto flex h-14 max-w-6xl items-center gap-6 px-5 sm:px-8">
        <a
          href={pathFor(locale)}
          className="flex items-center gap-2 font-mono text-base font-bold text-text"
        >
          <Logo size={24} className="rounded-md ring-1 ring-surface2" />
          <span className="rounded-md bg-accent px-2 py-0.5 text-base text-sm">gitpad</span>
        </a>
        <nav aria-label="Primary" className="hidden items-center gap-5 text-sm text-muted md:flex">
          {links.map(([href, label]) => (
            <a key={href} href={href} className="hover:text-text">
              {label}
            </a>
          ))}
        </nav>
        <div className="ml-auto flex items-center gap-3 text-sm">
          <a
            href={pathFor(other)}
            hrefLang={other}
            className="rounded-md border border-surface2 px-2.5 py-1 font-mono text-xs text-muted hover:border-accent hover:text-text"
            aria-label={other === "ko" ? "한국어로 보기" : "View in English"}
          >
            {locale === "en" ? "KO" : "EN"}
          </a>
          <a
            href={REPO}
            className="rounded-md bg-surface2 px-3 py-1.5 font-medium text-text hover:bg-overlay"
            target="_blank"
            rel="noreferrer"
          >
            {t.nav.github} ↗
          </a>
        </div>
      </div>
    </header>
  );
}
