import { useState } from "react";
import { useT } from "../i18n";

export function CopyButton({ text }: { text: string }) {
  const t = useT();
  const [done, setDone] = useState(false);
  return (
    <button
      type="button"
      onClick={() => {
        void navigator.clipboard?.writeText(text).then(() => {
          setDone(true);
          setTimeout(() => setDone(false), 1500);
        });
      }}
      className="shrink-0 rounded-md bg-surface2 px-2.5 py-1 text-xs font-medium text-muted hover:bg-overlay hover:text-text"
    >
      {done ? t.hero.copied : t.hero.copy}
    </button>
  );
}
