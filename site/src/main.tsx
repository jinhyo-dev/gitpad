import { StrictMode } from "react";
import { createRoot, hydrateRoot } from "react-dom/client";
import { App } from "./App";
import { localeFromPath } from "./i18n";
import "./styles.css";

const root = document.getElementById("app")!;
const locale = localeFromPath(window.location.pathname);
const tree = (
  <StrictMode>
    <App locale={locale} />
  </StrictMode>
);
if (root.hasChildNodes()) {
  hydrateRoot(root, tree);
} else {
  createRoot(root).render(tree);
}
