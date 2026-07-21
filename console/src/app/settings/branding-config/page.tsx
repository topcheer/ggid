"use client";
import { useState, useEffect } from "react";
import { Palette, Loader2, Save, Upload, Image as ImageIcon, Code } from "lucide-react";
import { useTranslations } from "@/lib/i18n";
import { useApi } from "@/lib/api";
import { DEFAULT_TENANT_ID } from "@/lib/api-config";

// This is a re-export of the branding page at /settings/branding-config
// to fix the 404 when navigating from the settings grid.
export default function BrandingConfigPage() {
  const t = useTranslations();
  const { apiFetch } = useApi();
  const [saving, setSaving] = useState(false);
  const [loaded, setLoaded] = useState(false);
  const [primaryColor, setPrimaryColor] = useState("#4f46e5");
  const [accentColor, setAccentColor] = useState("#06b6d4");
  const [fontFamily, setFontFamily] = useState("Inter");
  const [borderRadius, setBorderRadius] = useState(8);
  const [darkMode, setDarkMode] = useState(false);
  const [logoUrl, setLogoUrl] = useState("");
  const [customCss, setCustomCss] = useState("");
  const [msg, setMsg] = useState<string | null>(null);

  const card = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";
  const fonts = ["Inter", "Roboto", "Open Sans", "Lato", "Poppins", "Noto Sans SC"];

  useEffect(() => {
    const tid = localStorage.getItem("ggid_tenant_id") || DEFAULT_TENANT_ID;
    apiFetch<any>(`/api/v1/tenants/${tid}/branding`).then((b) => {
      if (b) {
        if (b.primary_color) setPrimaryColor(b.primary_color);
        if (b.accent_color) setAccentColor(b.accent_color);
        if (b.font_family) setFontFamily(b.font_family);
        if (b.border_radius) setBorderRadius(b.border_radius);
        if (b.default_mode === "dark") setDarkMode(true);
        if (b.logo_url) setLogoUrl(b.logo_url);
        if (b.custom_css) setCustomCss(b.custom_css);
      }
    }).catch(() => {}).finally(() => setLoaded(true));
  }, []);

  const save = async () => {
    setSaving(true);
    setMsg(null);
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
          logo_url: logoUrl,
          custom_css: customCss,
        }),
      });
      setMsg("Branding saved");
    } catch {
      setMsg("Save failed — API unavailable");
    }
    setSaving(false);
    setTimeout(() => setMsg(null), 3000);
  };

  if (!loaded) {
    return (
      <div className="flex items-center justify-center py-12">
        <Loader2 className="h-6 w-6 animate-spin text-brand-500" />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white dark:text-white">
          <Palette className="h-6 w-6 text-pink-500" /> Branding Configuration
        </h1>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
          Customize Console appearance, logo, and theme for your tenant.
        </p>
      </div>

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
        <div className={card}>
          <h3 className="mb-4 text-sm font-semibold uppercase text-gray-400">Color Palette</h3>
          <div className="space-y-4">
            <div>
              <label className="text-sm font-medium">Primary Color</label>
              <div className="mt-1 flex items-center gap-3">
                <input type="color" value={primaryColor} onChange={e => setPrimaryColor(e.target.value)} className="h-10 w-16 rounded cursor-pointer" />
                <code className="text-xs font-mono">{primaryColor}</code>
              </div>
            </div>
            <div>
              <label className="text-sm font-medium">Accent Color</label>
              <div className="mt-1 flex items-center gap-3">
                <input type="color" value={accentColor} onChange={e => setAccentColor(e.target.value)} className="h-10 w-16 rounded cursor-pointer" />
                <code className="text-xs font-mono">{accentColor}</code>
              </div>
            </div>
            <div>
              <label className="text-sm font-medium">Font Family</label>
              <select value={fontFamily} onChange={e => setFontFamily(e.target.value)} className="mt-1 block w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm">
                {fonts.map(f => <option key={f} value={f}>{f}</option>)}
              </select>
            </div>
            <div>
              <label className="text-sm font-medium">Border Radius</label>
              <div className="mt-1 flex items-center gap-3">
                <input type="range" min={0} max={20} value={borderRadius} onChange={e => setBorderRadius(parseInt(e.target.value))} className="flex-1 accent-pink-500" />
                <span className="text-sm font-mono w-8">{borderRadius}px</span>
              </div>
            </div>
          </div>
          <button onClick={save} disabled={saving} className="mt-4 flex items-center gap-2 rounded-lg bg-pink-600 px-4 py-2 text-sm font-medium text-white hover:bg-pink-700 disabled:opacity-50">
            {saving ? <Loader2 className="h-4 w-4 animate-spin" /> : <Save className="h-4 w-4" />} Save Theme
          </button>
          {msg && <p className="mt-2 text-sm text-green-600">{msg}</p>}
        </div>

        <div className={card}>
          <h3 className="mb-4 text-sm font-semibold uppercase text-gray-400">Logo & Assets</h3>
          <div className="space-y-4">
            <div>
              <label className="text-sm font-medium">Logo URL</label>
              <input type="text" value={logoUrl} onChange={e => setLogoUrl(e.target.value)} placeholder="https://example.com/logo.png" className="mt-1 block w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm" />
              <p className="mt-1 text-xs text-gray-400">SVG or PNG, max 512x512px</p>
            </div>
            <div className="flex items-center gap-4">
              <div className="flex h-16 w-16 items-center justify-center rounded-lg bg-gray-100 dark:bg-gray-700 dark:bg-gray-700 text-gray-300">
                {logoUrl ? <img src={logoUrl} alt="Logo" className="h-full w-full object-contain rounded-lg" /> : <ImageIcon className="h-8 w-8" />}
              </div>
            </div>
          </div>
          <div className="mt-4">
            <h4 className="mb-2 flex items-center gap-2 text-sm font-medium"><Code className="h-4 w-4" /> Custom CSS</h4>
            <textarea rows={6} value={customCss} onChange={e => setCustomCss(e.target.value)} placeholder={"/* Custom CSS injected into console */\n:root {\n  --brand-gradient: linear-gradient(...);\n}"} className="w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 font-mono text-xs" />
            <p className="mt-2 text-xs text-gray-400">Custom CSS is injected into the Console head for your tenant.</p>
          </div>
        </div>
      </div>
    </div>
  );
}