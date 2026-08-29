import { useT } from "../i18n";
import { Section } from "./Section";

/** Per-feature recordings (docs/media/*.gif copied into public/media). */
export function Gallery() {
  const t = useT();
  return (
    <Section id="gallery" title={t.gallery.title} subtitle={t.gallery.subtitle}>
      <div className="grid gap-6 md:grid-cols-2">
        {t.gallery.items.map((it, i) => (
          <figure
            key={it.id}
            className="overflow-hidden rounded-xl border border-surface2 bg-surface/60 transition hover:border-accent/60"
          >
            <img
              src={`/media/${it.id}.gif`}
              alt={it.title}
              loading={i < 2 ? "eager" : "lazy"}
              className="block w-full bg-base"
            />
            <figcaption className="p-5">
              <h3 className="text-lg font-semibold text-text">{it.title}</h3>
              <p className="mt-1.5 text-sm leading-relaxed text-muted">{it.body}</p>
            </figcaption>
          </figure>
        ))}
      </div>
    </Section>
  );
}
