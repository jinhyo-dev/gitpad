import { createContext, useContext } from "react";
import { en, type Dict } from "./en";
import { ko } from "./ko";

export type Locale = "en" | "ko";
export const locales: readonly Locale[] = ["en", "ko"] as const;
export const dicts: Record<Locale, Dict> = { en, ko };

export const BASE = import.meta.env.BASE_URL; // "/" unless `base` is configured

/** Locale from a pathname such as /gitpad/ko/ (defaults to English). */
export function localeFromPath(pathname: string): Locale {
  const rest = pathname.startsWith(BASE)
    ? pathname.slice(BASE.length)
    : pathname.replace(/^\//, "");
  const first = rest.split("/")[0];
  return first === "ko" ? "ko" : "en";
}

export function pathFor(locale: Locale): string {
  return `${BASE}${locale}/`;
}

export const LocaleContext = createContext<Locale>("en");

export function useLocale(): Locale {
  return useContext(LocaleContext);
}

export function useT(): Dict {
  return dicts[useContext(LocaleContext)];
}
