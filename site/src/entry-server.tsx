import { renderToString } from "react-dom/server";
import { App } from "./App";
import { dicts, type Locale } from "./i18n";

/** Used by scripts/prerender.ts to emit static /en/ and /ko/ pages. */
export function render(locale: Locale): { html: string; head: string } {
  const d = dicts[locale];
  const html = renderToString(<App locale={locale} />);
  const other: Locale = locale === "en" ? "ko" : "en";
  const head = [
    `<title>${d.meta.title}</title>`,
    `<meta name="description" content="${d.meta.description}" />`,
    `<meta property="og:title" content="${d.meta.title}" />`,
    `<meta property="og:description" content="${d.meta.description}" />`,
    `<meta property="og:type" content="website" />`,
    `<meta name="twitter:card" content="summary_large_image" />`,
    `<meta name="theme-color" content="#11111b" />`,
  ];
  // Absolute URLs only when the deployer tells us where the site lives.
  const site = (import.meta.env.VITE_SITE_URL as string | undefined)?.replace(/\/?$/, "/");
  if (site) {
    head.push(
      `<link rel="canonical" href="${site}${locale}/" />`,
      `<link rel="alternate" hreflang="${locale}" href="${site}${locale}/" />`,
      `<link rel="alternate" hreflang="${other}" href="${site}${other}/" />`,
      `<link rel="alternate" hreflang="x-default" href="${site}en/" />`,
      `<meta property="og:image" content="${site}og.png" />`,
    );
  }
  return { html, head: head.join("\n    ") };
}
