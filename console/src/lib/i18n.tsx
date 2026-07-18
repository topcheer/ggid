"use client";

import React, { createContext, useContext, useState, useEffect, useCallback } from "react";
import { en } from "./i18n-en";

// Dynamic locale cache — other languages loaded on demand
const localeCache: Record<string, Record<string, string>> = { en };

export type Locale =
  | "en" | "zh" | "zh-TW"
  | "es" | "hi" | "fr" | "ar" | "pt"
  | "ru" | "de" | "ja" | "ko"
  | "tr" | "vi" | "id";

type Dict = Record<string, string>;

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

/**
 * Locale metadata for UI selectors.
 */
export const LOCALE_INFO: Record<Locale, { label: string; flag: string; rtl?: boolean }> = {
  en:    { label: "English",    flag: "🇺🇸" },
  zh:    { label: "简体中文",    flag: "🇨🇳" },
  "zh-TW": { label: "繁體中文",  flag: "🇭🇰" },
  es:    { label: "Español",    flag: "🇪🇸" },
  hi:    { label: "हिन्दी",      flag: "🇮🇳" },
  fr:    { label: "Français",   flag: "🇫🇷" },
  ar:    { label: "العربية",     flag: "🇸🇦", rtl: true },
  pt:    { label: "Português",  flag: "🇵🇹" },
  ru:    { label: "Русский",    flag: "🇷🇺" },
  de:    { label: "Deutsch",    flag: "🇩🇪" },
  ja:    { label: "日本語",      flag: "🇯🇵" },
  ko:    { label: "한국어",      flag: "🇰🇷" },
  tr:    { label: "Türkçe",     flag: "🇹🇷" },
  vi:    { label: "Tiếng Việt", flag: "🇻🇳" },
  id:    { label: "Indonesia",  flag: "🇮🇩" },
};

const RTL_LOCALES = ["ar"];

export function I18nProvider({ children }: { children: React.ReactNode }) {
  const [locale, setLocaleState] = useState<Locale>("en");
  const [activeDict, setActiveDict] = useState<Record<string, string>>(en);

  // Load locale dict on demand
  useEffect(() => {
    const saved = localStorage.getItem("ggid_locale");
    const validLocales = Object.keys(LOCALE_INFO);
    if (saved && validLocales.includes(saved)) {
      setLocaleState(saved as Locale);
    }
  }, []);

  // When locale changes, load the dict if not cached
  useEffect(() => {
    if (locale === "en") {
      setActiveDict(en);
      return;
    }
    if (localeCache[locale]) {
      setActiveDict(localeCache[locale]);
      return;
    }
    // Dynamic import — only loads the requested language chunk
    import("./i18n-dicts").then((mod) => {
      const dict = mod.locales[locale];
      if (dict) {
        localeCache[locale] = dict;
        setActiveDict(dict);
      }
    }).catch(() => {
      // Fallback to English on load failure
      setActiveDict(en);
    });
  }, [locale]);

  const setLocale = useCallback((l: Locale) => {
    setLocaleState(l);
    localStorage.setItem("ggid_locale", l);
    const isRtl = RTL_LOCALES.includes(l);
    document.documentElement.dir = isRtl ? "rtl" : "ltr";
    document.documentElement.lang = l;
  }, []);

  const t = useCallback(
    (key: string): string => activeDict[key] || en[key] || key,
    [activeDict],
  );

  return <I18nContext.Provider value={{ locale, setLocale, t }}>{children}</I18nContext.Provider>;
}
