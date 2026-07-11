"use client";

import React, { createContext, useContext, useState, useEffect, useCallback } from "react";
import { en, zh } from "./i18n-dicts";

export type Locale = "en" | "zh";

type Dict = Record<string, string>;

const dictionaries: Record<Locale, Dict> = { en, zh };

interface I18nContextValue {
  locale: Locale;
  setLocale: (l: Locale) => void;
  t: (key: string) => string;
}

const I18nContext = createContext<I18nContextValue>({
  locale: "en",
  setLocale: () => {},
  t: (key: string) => key,
});

export function useI18n() {
  return useContext(I18nContext);
}

/**
 * useTranslations — convenience hook that returns a bound `t` function.
 * Usage: const t = useTranslations();  t("common.save")
 */
export function useTranslations() {
  const { t } = useContext(I18nContext);
  return t;
}

export function I18nProvider({ children }: { children: React.ReactNode }) {
  const [locale, setLocaleState] = useState<Locale>("en");

  useEffect(() => {
    const saved = localStorage.getItem("ggid_locale") as Locale;
    if (saved === "en" || saved === "zh") {
      setLocaleState(saved);
    }
  }, []);

  const setLocale = useCallback((l: Locale) => {
    setLocaleState(l);
    localStorage.setItem("ggid_locale", l);
  }, []);

  const t = useCallback(
    (key: string): string => dictionaries[locale]?.[key] || dictionaries.en[key] || key,
    [locale],
  );

  return <I18nContext.Provider value={{ locale, setLocale, t }}>{children}</I18nContext.Provider>;
}
