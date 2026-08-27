import { useT } from "../i18n";
import { Section } from "./Section";

const icons = ["⎇", "☰", "✎", "⇡", "✓", "⌕"];

export function Features() {
  const t = useT();
  return (
    <Section id="features" title={t.features.title} subtitle={t.features.subtitle}>
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {t.features.items.map((f, i) => (
          <div
            key={f.title}
            className="rounded-xl border border-surface2 bg-surface/60 p-5 transition hover:border-accent/60 hover:bg-surface"
          >
            <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-accent/15 font-mono text-lg text-accent">
              {icons[i]}
            </div>
            <h3 className="mt-4 text-lg font-semibold text-text">{f.title}</h3>
            <p className="mt-2 text-sm leading-relaxed text-muted">{f.body}</p>
          </div>
        ))}
      </div>
    </Section>
  );
}
