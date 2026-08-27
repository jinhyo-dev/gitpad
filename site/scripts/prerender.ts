// Emits static pages for every locale after `vite build`:
//   dist/index.html      → tiny redirect to /en/ or /ko/ by browser language
//   dist/en/index.html   → prerendered English page (hydrated on load)
//   dist/ko/index.html   → prerendered Korean page
//   dist/404.html        → same redirect (GitHub Pages fallback)
import { mkdirSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { pathToFileURL } from "node:url";

const BASE = "/";
const dist = new URL("../dist/", import.meta.url).pathname;
const template = readFileSync(dist + "index.html", "utf8");
const { render } = (await import(pathToFileURL(dist + "server/entry-server.js").href)) as {
  render: (locale: "en" | "ko") => { html: string; head: string };
};

for (const locale of ["en", "ko"] as const) {
  const { html, head } = render(locale);
  const page = template
    .replace('<html lang="en"', `<html lang="${locale}"`)
    .replace("<!--app-head-->", head)
    .replace("<!--app-html-->", html);
  mkdirSync(dist + locale, { recursive: true });
  writeFileSync(`${dist}${locale}/index.html`, page);
  console.log(`prerendered ${locale}/index.html (${(page.length / 1024).toFixed(0)} KB)`);
}

const redirect = `<!doctype html>
<html lang="en"><head><meta charset="utf-8"><title>gitpad</title>
<meta name="robots" content="noindex">
<script>
  var l = ((navigator.languages && navigator.languages[0]) || navigator.language || "en").toLowerCase();
  var target = "${BASE}" + (l.indexOf("ko") === 0 ? "ko" : "en") + "/" + location.hash;
  location.replace(target);
</script>
<style>body{font-family:system-ui;background:#11111b;color:#cdd6f4;display:grid;place-items:center;height:100vh;margin:0}</style>
</head><body><p>gitpad · <a href="${BASE}en/" style="color:#89b4fa">English</a> · <a href="${BASE}ko/" style="color:#89b4fa">한국어</a></p></body></html>
`;
writeFileSync(dist + "index.html", redirect);
writeFileSync(dist + "404.html", redirect);
console.log("wrote index.html + 404.html redirects");
rmSync(dist + "server", { recursive: true, force: true }); // SSR bundle is only needed here
