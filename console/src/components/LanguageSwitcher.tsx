"use client";

import { useState, useRef, useEffect } from "react";
import { useI18n, LOCALE_INFO, type Locale } from "@/lib/i18n";
import { Globe, Check, ChevronDown } from "lucide-react";

/**
 * Language switcher dropdown supporting all available locales.
 * Persists choice in localStorage via useI18n.
 *
 * Usage: <LanguageSwitcher /> or <LanguageSwitcher compact />
 */
export function LanguageSwitcher({ compact = false }: { compact?: boolean }) {
  const { locale, setLocale } = useI18n();
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    function handleClickOutside(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        setOpen(false);
      }
    }
    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, []);

  const allLocales = Object.keys(LOCALE_INFO) as Locale[];
  const current = LOCALE_INFO[locale];

  if (compact) {
    return (
      <div ref={ref} className="relative">
        <button
          onClick={() => setOpen(!open)}
          className="flex items-center gap-1.5 rounded px-2 py-1 text-sm font-medium hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors"
          aria-label="Switch language"
        >
          <span>{current?.flag}</span>
          <span className="hidden sm:inline">{locale.toUpperCase()}</span>
          <ChevronDown className="h-3 w-3 opacity-50" />
        </button>
        {open && (
          <div className="absolute right-0 z-50 mt-1 max-h-80 overflow-y-auto rounded-lg border border-gray-200 bg-white py-1 shadow-lg dark:border-gray-700 dark:bg-gray-900">
            {allLocales.map((l) => {
              const info = LOCALE_INFO[l];
              return (
                <button
                  key={l}
                  onClick={() => { setLocale(l); setOpen(false); }}
                  className={`flex w-full items-center gap-2 px-3 py-1.5 text-sm transition-colors ${
                    locale === l
                      ? "bg-blue-50 text-blue-700 dark:bg-blue-950 dark:text-blue-400"
                      : "text-gray-700 hover:bg-gray-50 dark:text-gray-300 dark:hover:bg-gray-800"
                  }`}
                >
                  <span className="text-base">{info.flag}</span>
                  <span className="flex-1 text-left">{info.label}</span>
                  {locale === l && <Check className="h-3.5 w-3.5" />}
                </button>
              );
            })}
          </div>
        )}
      </div>
    );
  }

  return (
    <div ref={ref} className="relative">
      <button
        onClick={() => setOpen(!open)}
        className="flex items-center gap-2 rounded-lg border border-gray-200 px-3 py-1.5 text-sm font-medium transition-colors hover:bg-gray-50 dark:border-gray-700 dark:hover:bg-gray-800"
        aria-label="Switch language"
      >
        <Globe className="h-4 w-4 text-gray-400" />
        <span>{current?.flag}</span>
        <span>{current?.label}</span>
        <ChevronDown className="h-3 w-3 opacity-50" />
      </button>
      {open && (
        <div className="absolute right-0 z-50 mt-1 max-h-80 overflow-y-auto rounded-lg border border-gray-200 bg-white py-1 shadow-lg dark:border-gray-700 dark:bg-gray-900">
          {allLocales.map((l) => {
            const info = LOCALE_INFO[l];
            return (
              <button
                key={l}
                onClick={() => { setLocale(l); setOpen(false); }}
                className={`flex w-full items-center gap-2 px-3 py-2 text-sm transition-colors ${
                  locale === l
                    ? "bg-blue-50 text-blue-700 dark:bg-blue-950 dark:text-blue-400"
                    : "text-gray-700 hover:bg-gray-50 dark:text-gray-300 dark:hover:bg-gray-800"
                }`}
              >
                <span className="text-base">{info.flag}</span>
                <span className="flex-1 text-left">{info.label}</span>
                {info.rtl && <span className="text-xs text-gray-400">RTL</span>}
                {locale === l && <Check className="h-3.5 w-3.5" />}
              </button>
            );
          })}
        </div>
      )}
    </div>
  );
}
