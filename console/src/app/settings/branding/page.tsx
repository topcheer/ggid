"use client";
import { useState, useEffect } from "react";
import {
  Palette, Loader2, AlertCircle, X, Upload, Check, Eye, Mail,
  Save, Type, Square, Moon, Sun, Image, Code, ChevronRight,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";
import { useApi } from "@/lib/api";
import { DEFAULT_TENANT_ID } from "@/lib/api-config";

type Tab = "theme" | "assets" | "email";

export default function BrandingPage() {
  const t = useTranslations();
  const { apiFetch } = useApi();
  const [tab, setTab] = useState<Tab>("theme");
  const [saving, setSaving] = useState(false);
  const [loaded, setLoaded] = useState(false);

  // Theme
  const [primaryColor, setPrimaryColor] = useState("#4f46e5");
  const [accentColor, setAccentColor] = useState("#06b6d4");
  const [fontFamily, setFontFamily] = useState("Inter");
  const [borderRadius, setBorderRadius] = useState(8);
  const [darkMode, setDarkMode] = useState(false);

  const card = "rounded-xl border border-gray-200 bg-white dark:border-gray-700 dark:bg-gray-800 p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";
  const fonts = ["Inter", "Roboto", "Open Sans", "Lato", "Poppins", "Noto Sans SC"];

  // Load branding from API on mount
  useEffect(() => {
    const tid = localStorage.getItem("ggid_tenant_id") || DEFAULT_TENANT_ID;
    apiFetch<any>(`/api/v1/tenants/${tid}/branding`).then((b) => {
      if (b) {
        if (b.primary_color) setPrimaryColor(b.primary_color);
        if (b.accent_color) setAccentColor(b.accent_color);
        if (b.font_family) setFontFamily(b.font_family);
        if (b.border_radius) setBorderRadius(b.border_radius);
        if (b.default_mode === "dark") setDarkMode(true);
      }
    }).catch(() => {}).finally(() => setLoaded(true));
  }, []);

  const save = async () => {
    setSaving(true);
    const tid = localStorage.getItem("ggid_tenant_id") || DEFAULT_TENANT_ID;
    try {
      await apiFetch(`/api/v1/tenants/${tid}/branding`, {
        method: "PUT",
        body: JSON.stringify({
          primary_color: primaryColor,
          accent_color: accentColor,
          font_family: fontFamily,
          border_radius: borderRadius,
          default_mode: darkMode ? "dark" : "light",
        }),
      });
    } catch { /* ok for demo */ }
    setSaving(false);
  };

  return (
    <div className="space-y-6">
      <div><h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white dark:text-white"><Palette className="h-6 w-6 text-pink-500" /> {t("branding.title")}</h1><p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{t("branding.subtitle")}</p></div>

      <div className="flex gap-1 border-b border-gray-200 dark:border-gray-700 dark:border-gray-700 overflow-x-auto">
        {([["theme", t("branding.theme"), Palette], ["assets", t("branding.assets"), Image], ["email", t("branding.emailTemplates"), Mail]] as const).map(([id, label, Icon]) => (
          <button key={id} onClick={() => setTab(id as Tab)} aria-pressed={tab === id} className={`flex items-center gap-1.5 border-b-2 px-4 py-2 text-sm font-medium transition whitespace-nowrap ${tab === id ? "border-pink-600 text-pink-600 dark:text-pink-400" : "border-transparent text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"}`}><Icon className="h-4 w-4" /> {label}</button>
        ))}
      </div>

      {/* THEME */}
      {tab === "theme" && (
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
          <div className={card}>
            <h3 className="mb-4 text-sm font-semibold uppercase text-gray-400">{t("branding.colorPalette")}</h3>
            <div className="space-y-4">
              <div><label className="text-sm font-medium">{t("branding.primaryColor")}</label><div className="mt-1 flex items-center gap-3"><input type="color" value={primaryColor} onChange={e => setPrimaryColor(e.target.value)} className="h-10 w-16 rounded cursor-pointer" /><code className="text-xs font-mono">{primaryColor}</code></div></div>
              <div><label className="text-sm font-medium">{t("branding.accentColor")}</label><div className="mt-1 flex items-center gap-3"><input type="color" value={accentColor} onChange={e => setAccentColor(e.target.value)} className="h-10 w-16 rounded cursor-pointer" /><code className="text-xs font-mono">{accentColor}</code></div></div>
              <div><label className="text-sm font-medium flex items-center gap-2"><Type className="h-4 w-4" /> {t("branding.fontFamily")}</label><select value={fontFamily} onChange={e => setFontFamily(e.target.value)} className="mt-1 block w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm">{fonts.map(f => <option key={f} value={f}>{f}</option>)}</select></div>
              <div><label className="text-sm font-medium flex items-center gap-2"><Square className="h-4 w-4" /> {t("branding.borderRadius")}</label><div className="mt-1 flex items-center gap-3"><input type="range" min={0} max={20} value={borderRadius} onChange={e => setBorderRadius(parseInt(e.target.value))} className="flex-1 accent-pink-500" /><span className="text-sm font-mono w-8">{borderRadius}px</span></div></div>
              <div><label className="text-sm font-medium flex items-center gap-2">{darkMode ? <Moon className="h-4 w-4" /> : <Sun className="h-4 w-4" />} {t("branding.defaultMode")}</label><div className="mt-1 flex gap-2"><button onClick={() => setDarkMode(false)} aria-pressed={!darkMode} className={`flex-1 rounded-lg border px-3 py-2 text-sm ${!darkMode ? "border-pink-500 bg-pink-50 dark:bg-pink-950/30 text-pink-600" : "border-gray-300 dark:border-gray-700"}`}><Sun className="inline h-4 w-4" /> Light</button><button onClick={() => setDarkMode(true)} aria-pressed={darkMode} className={`flex-1 rounded-lg border px-3 py-2 text-sm ${darkMode ? "border-pink-500 bg-pink-50 dark:bg-pink-950/30 text-pink-600" : "border-gray-300 dark:border-gray-700"}`}><Moon className="inline h-4 w-4" /> Dark</button></div></div>
            </div>
            <button onClick={save} disabled={saving} className="mt-4 flex items-center gap-2 rounded-lg bg-pink-600 px-4 py-2 text-sm font-medium text-white hover:bg-pink-700 disabled:opacity-50">{saving ? <Loader2 className="h-4 w-4 animate-spin" /> : <Save className="h-4 w-4" />} {t("branding.saveTheme")}</button>
          </div>
          {/* Live preview */}
          <div className={card}>
            <h3 className="mb-3 text-sm font-semibold uppercase text-gray-400">{t("branding.livePreview")}</h3>
            <div className={`rounded-xl border-2 p-6 transition ${darkMode ? "bg-gray-900 border-gray-700" : "bg-white border-gray-200 dark:border-gray-700"}`} style={{ borderRadius: `${borderRadius}px` }}>
              <div className="text-center mb-4"><div className="mx-auto flex h-12 w-12 items-center justify-center rounded-lg font-bold text-white" style={{ backgroundColor: primaryColor, borderRadius: `${borderRadius}px` }}>GG</div><h3 className={`mt-2 text-lg font-bold ${darkMode ? "text-white" : "text-gray-900"}`} style={{ fontFamily }}>{t("branding.welcomeBack")}</h3><p className={`text-xs ${darkMode ? "text-gray-400" : "text-gray-500 dark:text-gray-400"}`}>{t("branding.signInPrompt")}</p></div>
              <div className="space-y-2"><div className={`h-10 rounded-lg border ${darkMode ? "border-gray-700 bg-gray-800" : "border-gray-200 dark:border-gray-700"} px-3 flex items-center text-xs ${darkMode ? "text-gray-400" : "text-gray-400"}`} style={{ borderRadius: `${borderRadius}px` }}>{t("branding.emailPlaceholder")}</div><div className={`h-10 rounded-lg border ${darkMode ? "border-gray-700 bg-gray-800" : "border-gray-200 dark:border-gray-700"} px-3 flex items-center text-xs ${darkMode ? "text-gray-400" : "text-gray-400"}`} style={{ borderRadius: `${borderRadius}px` }}>{t("branding.passwordPlaceholder")}</div><button className="w-full h-10 text-white text-sm font-medium flex items-center justify-center" style={{ backgroundColor: primaryColor, borderRadius: `${borderRadius}px`, fontFamily }}>{t("branding.signIn")}</button></div>
              <div className="mt-3 text-center"><a href="#" className="text-xs" style={{ color: accentColor }}>{t("branding.forgotPassword")}</a></div>
            </div>
          </div>
        </div>
      )}

      {/* ASSETS */}
      {tab === "assets" && (
        <div className="space-y-6">
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <div className={card}>
              <h3 className="mb-3 text-sm font-semibold">{t("branding.logo")}</h3>
              <div className="flex items-center gap-4"><div className="flex h-16 w-16 items-center justify-center rounded-lg bg-gray-100 dark:bg-gray-700 dark:bg-gray-700 text-gray-300"><Image className="h-8 w-8" /></div><button className="rounded-lg border border-gray-300 dark:border-gray-600 px-3 py-2 text-sm dark:border-gray-700 flex items-center gap-2"><Upload className="h-3.5 w-3.5" /> {t("branding.upload")}</button></div>
              <p className="mt-2 text-xs text-gray-400">SVG, PNG — max 512×512px</p>
            </div>
            <div className={card}>
              <h3 className="mb-3 text-sm font-semibold">{t("branding.favicon")}</h3>
              <div className="flex items-center gap-4"><div className="flex h-16 w-16 items-center justify-center rounded-lg bg-gray-100 dark:bg-gray-700 dark:bg-gray-700 text-gray-300"><Image className="h-8 w-8" /></div><button className="rounded-lg border border-gray-300 dark:border-gray-600 px-3 py-2 text-sm dark:border-gray-700 flex items-center gap-2"><Upload className="h-3.5 w-3.5" /> {t("branding.upload")}</button></div>
              <p className="mt-2 text-xs text-gray-400">ICO, PNG — 32×32px / 16×16px</p>
            </div>
          </div>
          <div className={card}>
            <h3 className="mb-3 flex items-center gap-2 text-sm font-semibold"><Code className="h-4 w-4" /> {t("branding.customCss")}</h3>
            <textarea rows={8} placeholder={"/* Custom CSS injected into console */\n:root {\n  --brand-gradient: linear-gradient(...);\n}"} className="w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 font-mono text-xs" />
            <p className="mt-2 text-xs text-gray-400">{t("branding.customCssNote")}</p>
          </div>
        </div>
      )}

      {/* EMAIL */}
      {tab === "email" && (
        <div className="space-y-4">
          {["verification", "passwordReset", "welcome"].map(type => (
            <div key={type} className={card}>
              <div className="flex items-center justify-between mb-3"><div className="flex items-center gap-3"><Mail className="h-5 w-5 text-pink-400" /><div><h3 className="font-semibold text-sm capitalize">{type.replace(/([A-Z])/g, " $1")}</h3><p className="text-xs text-gray-400">{t(`branding.${type}Desc`)}</p></div></div><button className="flex items-center gap-1 rounded-lg border border-gray-300 dark:border-gray-600 px-3 py-1.5 text-xs dark:border-gray-700"><Eye className="h-3 w-3" /> {t("branding.previewEmail")}</button></div>
              <div className="rounded-lg border p-4 dark:border-gray-700" style={{ borderRadius: `${borderRadius}px` }}>
                <div className="rounded-t-lg p-4 text-white text-center" style={{ backgroundColor: primaryColor, borderRadius: `${borderRadius}px ${borderRadius}px 0 0` }}><div className="mx-auto flex h-8 w-8 items-center justify-center rounded font-bold" style={{ fontFamily }}>GG</div></div>
                <div className="bg-white dark:bg-gray-800 dark:bg-gray-900 p-4"><h4 className={`font-semibold text-sm ${darkMode ? "text-white" : "text-gray-900"}`} style={{ fontFamily }}>{t(`branding.${type}Subject`)}</h4><p className={`mt-1 text-xs ${darkMode ? "text-gray-400" : "text-gray-500 dark:text-gray-400"}`}>{t(`branding.${type}Body`)}</p><button className="mt-3 px-4 py-2 text-white text-xs font-medium" style={{ backgroundColor: primaryColor, borderRadius: `${borderRadius}px` }}>{t(`branding.${type}Button`)}</button></div>
                <div className="bg-gray-50 dark:bg-gray-800 dark:bg-gray-800 p-2 text-center text-xs text-gray-400">© 2025 GGID · {t("branding.footerText")}</div>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
