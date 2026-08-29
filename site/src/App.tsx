import { useEffect } from "react";
import { dicts, LocaleContext, type Locale } from "./i18n";
import { Header } from "./components/Header";
import { Hero } from "./components/Hero";
import { Demo } from "./demo/Demo";
import { About } from "./components/About";
import { Gallery } from "./components/Gallery";
import { Features } from "./components/Features";
import { Install } from "./components/Install";
import { Keys } from "./components/Keys";
import { Footer } from "./components/Footer";

export function App({ locale }: { locale: Locale }) {
  // Prerendered pages ship these in <head>; the dev server and client-side
  // navigation need them set here.
  useEffect(() => {
    const d = dicts[locale];
    document.title = d.meta.title;
    document.documentElement.lang = locale;
    let meta = document.querySelector<HTMLMetaElement>('meta[name="description"]');
    if (!meta) {
      meta = document.createElement("meta");
      meta.name = "description";
      document.head.appendChild(meta);
    }
    meta.content = d.meta.description;
  }, [locale]);
  return (
    <LocaleContext.Provider value={locale}>
      <div className="min-h-screen font-sans">
        <Header />
        <main>
          <Hero />
          <Demo />
          <About />
          <Gallery />
          <Features />
          <Install />
          <Keys />
        </main>
        <Footer />
      </div>
    </LocaleContext.Provider>
  );
}
