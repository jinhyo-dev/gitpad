import { LocaleContext, type Locale } from "./i18n";
import { Header } from "./components/Header";
import { Hero } from "./components/Hero";
import { Demo } from "./demo/Demo";
import { Features } from "./components/Features";
import { Install } from "./components/Install";
import { Keys } from "./components/Keys";
import { Footer } from "./components/Footer";

export function App({ locale }: { locale: Locale }) {
  return (
    <LocaleContext.Provider value={locale}>
      <div className="min-h-screen font-sans">
        <Header />
        <main>
          <Hero />
          <Demo />
          <Features />
          <Install />
          <Keys />
        </main>
        <Footer />
      </div>
    </LocaleContext.Provider>
  );
}
