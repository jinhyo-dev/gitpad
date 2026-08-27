import { useT } from "../i18n";
import { Section } from "./Section";

/** Plain-language overview — also the page's keyword-bearing copy for search. */
export function About() {
  const t = useT();
  return (
    <Section id="about" title={t.about.title}>
      <div className="grid gap-8 lg:grid-cols-[2fr_1fr]">
        <div className="space-y-4 text-lg leading-relaxed text-muted">
          {t.about.paragraphs.map((p) => (
            <p key={p}>{p}</p>
          ))}
        </div>
        <dl className="h-fit rounded-xl border border-surface2 bg-surface/60 p-5 text-sm">
          {t.about.facts.map(([k, v]) => (
            <div
              key={k}
              className="flex justify-between gap-4 border-b border-surface2/60 py-2 last:border-0"
            >
              <dt className="text-muted">{k}</dt>
              <dd className="text-right font-medium text-text">{v}</dd>
            </div>
          ))}
        </dl>
      </div>
    </Section>
  );
}
