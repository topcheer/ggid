"use client";

import { useI18n, type Locale } from "@/lib/i18n";

/**
 * Standalone language switcher button.
 * Toggles between EN and ZH. Persists choice in localStorage.
 *
 * Usage: <LanguageSwitcher /> or <LanguageSwitcher compact />
 */
export function LanguageSwitcher({ compact = false }: { compact?: boolean }) {
  const { locale, setLocale } = useI18n();

  const toggle = () => {
    const next: Locale = locale === "en" ? "zh" : "en";
    setLocale(next);
  };

  if (compact) {
    return (
      <button
        onClick={toggle}
        className="px-2 py-1 rounded text-sm font-medium hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors"
        title={locale === "en" ? "Switch to Chinese" : "\u5207\u6362\u5230\u82f1\u8bed"}
      >
        {locale === "en" ? "EN" : "\u4e2d"}
      </button>
    );
  }

  return (
    <div className="inline-flex items-center rounded-lg border border-gray-200 dark:border-gray-700 overflow-hidden">
      {(["en", "zh"] as Locale[]).map((l) => (
        <button
          key={l}
          onClick={() => setLocale(l)}
          className={`px-3 py-1.5 text-sm font-medium transition-colors ${
            locale === l
              ? "bg-blue-600 text-white"
              : "text-gray-600 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-gray-800"
          }`}
        >
          {l === "en" ? "English" : "\u4e2d\u6587"}
        </button>
      ))}
    </div>
  );
}
