import type { ReactNode } from "react";

export function Section({
  id,
  title,
  subtitle,
  children,
}: {
  id: string;
  title: string;
  subtitle?: string;
  children: ReactNode;
}) {
  return (
    <section id={id} className="mx-auto max-w-6xl scroll-mt-24 px-5 py-20 sm:px-8">
      <h2 className="text-3xl font-bold tracking-tight text-text sm:text-4xl">{title}</h2>
      {subtitle && <p className="mt-3 max-w-2xl text-lg text-muted">{subtitle}</p>}
      <div className="mt-10">{children}</div>
    </section>
  );
}
