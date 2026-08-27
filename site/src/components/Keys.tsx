import { useT } from "../i18n";
import { Section } from "./Section";

export function Keys() {
  const t = useT();
  return (
    <Section id="keys" title={t.keys.title} subtitle={t.keys.subtitle}>
      <div className="grid gap-4 sm:grid-cols-2">
        {t.keys.groups.map((g) => (
          <div key={g.title} className="rounded-xl border border-surface2 bg-surface/60 p-5">
            <h3 className="font-semibold text-text">{g.title}</h3>
            <table className="mt-3 w-full text-sm">
              <tbody>
                {g.rows.map(([k, d]) => (
                  <tr key={k} className="border-t border-surface2/60">
                    <td className="w-52 py-2 pr-3 align-top">
                      <span className="kbd whitespace-nowrap">{k}</span>
                    </td>
                    <td className="py-2 text-muted">{d}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ))}
      </div>
    </Section>
  );
}
