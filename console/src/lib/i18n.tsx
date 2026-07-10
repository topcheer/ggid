"use client";

import { useState, createContext, useContext } from "react";

export type Locale = "en" | "zh";

type Dict = Record<string, string>;

const en: Dict = {
  // Sidebar / nav
  "nav.dashboard": "Dashboard",
  "nav.users": "Users",
  "nav.roles": "Roles & Permissions",
  "nav.organizations": "Organizations",
  "nav.audit": "Audit Log",
  "nav.oauthClients": "OAuth Clients",
  "nav.webhooks": "Webhooks",
  "nav.settings": "Settings",
  // Common
  "common.loading": "Loading...",
  "common.create": "Create",
  "common.delete": "Delete",
  "common.cancel": "Cancel",
  "common.save": "Save",
  "common.edit": "Edit",
  "common.search": "Search",
  "common.refresh": "Refresh",
  "common.export": "Export",
  // Users page
  "users.title": "Users",
  "users.searchPlaceholder": "Search by name or email...",
  // Roles page
  "roles.title": "Roles & Permissions",
  // Orgs page
  "orgs.title": "Organizations",
  // Audit page
  "audit.title": "Audit Log",
  "audit.dashboard": "Dashboard",
  "audit.events": "Event Log",
  // OAuth clients
  "oauth.title": "OAuth Clients",
  // Webhooks
  "webhooks.title": "Webhooks",
  // Settings
  "settings.title": "Settings",
  "settings.theme": "Theme",
  "settings.language": "Language",
  "settings.darkMode": "Dark Mode",
};

const zh: Dict = {
  // Sidebar / nav
  "nav.dashboard": "\u4eea\u8868\u76d8",
  "nav.users": "\u7528\u6237",
  "nav.roles": "\u89d2\u8272\u4e0e\u6743\u9650",
  "nav.organizations": "\u7ec4\u7ec7\u67b6\u6784",
  "nav.audit": "\u5ba1\u8ba1\u65e5\u5fd7",
  "nav.oauthClients": "OAuth \u5ba2\u6237\u7aef",
  "nav.webhooks": "Webhook",
  "nav.settings": "\u8bbe\u7f6e",
  // Common
  "common.loading": "\u52a0\u8f7d\u4e2d...",
  "common.create": "\u521b\u5efa",
  "common.delete": "\u5220\u9664",
  "common.cancel": "\u53d6\u6d88",
  "common.save": "\u4fdd\u5b58",
  "common.edit": "\u7f16\u8f91",
  "common.search": "\u641c\u7d22",
  "common.refresh": "\u5237\u65b0",
  "common.export": "\u5bfc\u51fa",
  // Users page
  "users.title": "\u7528\u6237\u7ba1\u7406",
  "users.searchPlaceholder": "\u6309\u59d3\u540d\u6216\u90ae\u7bb1\u641c\u7d22...",
  // Roles page
  "roles.title": "\u89d2\u8272\u4e0e\u6743\u9650",
  // Orgs page
  "orgs.title": "\u7ec4\u7ec7\u67b6\u6784",
  // Audit page
  "audit.title": "\u5ba1\u8ba1\u65e5\u5fd7",
  "audit.dashboard": "\u4eea\u8868\u76d8",
  "audit.events": "\u4e8b\u4ef6\u65e5\u5fd7",
  // OAuth clients
  "oauth.title": "OAuth \u5ba2\u6237\u7aef",
  // Webhooks
  "webhooks.title": "Webhook",
  // Settings
  "settings.title": "\u8bbe\u7f6e",
  "settings.theme": "\u4e3b\u9898",
  "settings.language": "\u8bed\u8a00",
  "settings.darkMode": "\u6697\u8272\u6a21\u5f0f",
};

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

import { useEffect } from "react";

export function I18nProvider({ children }: { children: React.ReactNode }) {
  const [locale, setLocaleState] = useState<Locale>("en");

  useEffect(() => {
    const saved = localStorage.getItem("ggid_locale") as Locale;
    if (saved === "en" || saved === "zh") {
      setLocaleState(saved);
    }
  }, []);

  const setLocale = (l: Locale) => {
    setLocaleState(l);
    localStorage.setItem("ggid_locale", l);
  };

  const t = (key: string): string => {
    return dictionaries[locale][key] || dictionaries.en[key] || key;
  };

  return <I18nContext.Provider value={{ locale, setLocale, t }}>{children}</I18nContext.Provider>;
}
