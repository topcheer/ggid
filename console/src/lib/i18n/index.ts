// i18n lazy loader — imports only the requested language

const loaders: Record<string, () => Promise<Record<string, string>>> = {
  ar: () => import("./ar").then(m => m.ar),
  de: () => import("./de").then(m => m.de),
  en: () => import("./en").then(m => m.en),
  es: () => import("./es").then(m => m.es),
  fr: () => import("./fr").then(m => m.fr),
  hi: () => import("./hi").then(m => m.hi),
  id: () => import("./id").then(m => m.id),
  ja: () => import("./ja").then(m => m.ja),
  ko: () => import("./ko").then(m => m.ko),
  pt: () => import("./pt").then(m => m.pt),
  ru: () => import("./ru").then(m => m.ru),
  tr: () => import("./tr").then(m => m.tr),
  vi: () => import("./vi").then(m => m.vi),
  zh: () => import("./zh").then(m => m.zh),
  "zh-TW": () => import("./zh_TW").then(m => m.zh_TW),
};

export async function loadLocale(locale: string): Promise<Record<string, string> | null> {
  const loader = loaders[locale];
  if (!loader) return null;
  return loader();
}
