import { renderToString } from "react-dom/server";
import { App } from "./App";
import { dicts, type Locale } from "./i18n";

const REPO = "https://github.com/jinhyo-dev/gitpad";

function esc(s: string): string {
  return s.replace(/&/g, "&amp;").replace(/"/g, "&quot;").replace(/</g, "&lt;");
}

/** Used by scripts/prerender.ts to emit static /en/ and /ko/ pages. */
export function render(locale: Locale): { html: string; head: string } {
  const d = dicts[locale];
  const html = renderToString(<App locale={locale} />);
  const other: Locale = locale === "en" ? "ko" : "en";
  // Absolute URLs only when the deployer tells us where the site lives.
  const site = (import.meta.env.VITE_SITE_URL as string | undefined)?.replace(/\/?$/, "/");
  const pageUrl = site ? `${site}${locale}/` : undefined;

  const head: string[] = [
    `<title>${esc(d.meta.title)}</title>`,
    `<meta name="description" content="${esc(d.meta.description)}" />`,
    `<meta name="keywords" content="${esc(d.meta.keywords.join(", "))}" />`,
    `<meta name="robots" content="index,follow,max-image-preview:large" />`,
    `<meta name="author" content="jinhyo-dev" />`,
    `<meta name="application-name" content="gitpad" />`,
    `<meta name="theme-color" content="#11111b" />`,
    `<meta property="og:type" content="website" />`,
    `<meta property="og:site_name" content="${esc(d.meta.siteName)}" />`,
    `<meta property="og:title" content="${esc(d.meta.title)}" />`,
    `<meta property="og:description" content="${esc(d.meta.description)}" />`,
    `<meta property="og:locale" content="${d.meta.ogLocale}" />`,
    `<meta property="og:locale:alternate" content="${dicts[other].meta.ogLocale}" />`,
    `<meta name="twitter:card" content="summary_large_image" />`,
    `<meta name="twitter:title" content="${esc(d.meta.title)}" />`,
    `<meta name="twitter:description" content="${esc(d.meta.description)}" />`,
  ];
  if (site) {
    head.push(
      `<link rel="canonical" href="${pageUrl}" />`,
      `<link rel="alternate" hreflang="${locale}" href="${pageUrl}" />`,
      `<link rel="alternate" hreflang="${other}" href="${site}${other}/" />`,
      `<link rel="alternate" hreflang="x-default" href="${site}en/" />`,
      `<meta property="og:url" content="${pageUrl}" />`,
      `<meta property="og:image" content="${site}og.png" />`,
      `<meta property="og:image:width" content="1440" />`,
      `<meta property="og:image:height" content="920" />`,
      `<meta property="og:image:alt" content="${esc(d.meta.title)}" />`,
      `<meta name="twitter:image" content="${site}og.png" />`,
    );
  }

  // Structured data: the app itself, plus the website with its language.
  const app = {
    "@context": "https://schema.org",
    "@type": "SoftwareApplication",
    name: "gitpad",
    description: d.meta.description,
    applicationCategory: "DeveloperApplication",
    applicationSubCategory: "Git client",
    operatingSystem: "macOS, Linux, Windows",
    softwareRequirements: "git",
    license: `${REPO}/blob/main/LICENSE`,
    codeRepository: REPO,
    downloadUrl: `${REPO}/releases`,
    isAccessibleForFree: true,
    offers: { "@type": "Offer", price: "0", priceCurrency: "USD" },
    author: { "@type": "Person", name: "jinhyo-dev", url: "https://github.com/jinhyo-dev" },
    keywords: d.meta.keywords.join(", "),
    inLanguage: locale,
    ...(pageUrl ? { url: pageUrl, image: `${site}og.png` } : {}),
  };
  const website = pageUrl
    ? {
        "@context": "https://schema.org",
        "@type": "WebSite",
        name: "gitpad",
        url: site,
        inLanguage: [locale, other],
      }
    : null;
  head.push(`<script type="application/ld+json">${JSON.stringify(app)}</script>`);
  if (website) head.push(`<script type="application/ld+json">${JSON.stringify(website)}</script>`);

  return { html, head: head.join("\n    ") };
}
