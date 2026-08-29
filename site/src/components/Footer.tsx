import { pathFor, useLocale, useT } from "../i18n";
import { Logo } from "./Logo";

export function Footer() {
  const t = useT();
  const locale = useLocale();
  const other = locale === "en" ? "ko" : "en";
  return (
    <footer className="border-t border-surface2/60 py-10">
      <div className="mx-auto flex max-w-6xl flex-wrap items-center gap-x-6 gap-y-3 px-5 text-sm text-muted sm:px-8">
        <span className="flex items-center gap-2 font-mono font-bold text-text">
          <Logo size={18} className="rounded" />
          gitpad
        </span>
        <span className="ml-auto flex gap-5">
          <a
            href="https://github.com/jinhyo-dev/gitpad"
            className="hover:text-text"
            target="_blank"
            rel="noreferrer"
          >
            {t.footer.source}
          </a>
          <a
            href="https://github.com/jinhyo-dev/gitpad/blob/main/LICENSE"
            className="hover:text-text"
            target="_blank"
            rel="noreferrer"
          >
            {t.footer.license}
          </a>
          <a href={pathFor(other)} hrefLang={other} className="hover:text-text">
            {t.footer.lang}
          </a>
        </span>
        <div className="mx-auto mt-6 max-w-6xl px-5 text-xs text-dim sm:px-8">
          {t.footer.copyright}
        </div>
      </div>
    </footer>
  );
}
