"use client";

import { useState, useCallback, useRef } from "react";
import { useTranslations } from "@/lib/i18n";
import { authHeader } from "@/lib/auth-helpers";
import {
  Palette, Upload, Check, Loader2, RotateCcw, Eye, Save,
  AlertCircle, Shield, Image as ImageIcon,
} from "lucide-react";

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "";

interface BrandingConfig {
  app_name: string; logo_url: string;
  primary_color: string; secondary_color: string; accent_color: string;
  login_subtitle: string; footer_text: string;
}

const DEFAULTS: BrandingConfig = {
  app_name: "GGID", logo_url: "",
  primary_color: "#4f46e5", secondary_color: "#7c3aed", accent_color: "#06b6d4",
  login_subtitle: "Sign in to your account", footer_text: "Powered by GGID",
};

const PRESETS = [
  { name: "Ocean Blue", primary: "#2563eb", secondary: "#3b82f6", accent: "#06b6d4" },
  { name: "Forest Green", primary: "#059669", secondary: "#10b981", accent: "#84cc16" },
  { name: "Royal Purple", primary: "#7c3aed", secondary: "#a855f7", accent: "#ec4899" },
  { name: "Sunset Orange", primary: "#ea580c", secondary: "#f97316", accent: "#facc15" },
];

export default function BrandingConfigPage() {
  const t = useTranslations();
  const [config, setConfig] = useState<BrandingConfig>(DEFAULTS);
  const [saving, setSaving] = useState(false);
  const [msg, setMsg] = useState<string | null>(null);
  const [logoPreview, setLogoPreview] = useState("");
  const fileRef = useRef<HTMLInputElement>(null);

  const load = useCallback(async () => {
    try {
      const res = await fetch(`${API_BASE}/api/v1/identity/branding/config`, { headers: { ...authHeader() } });
      if (res.ok) { const d = await res.json(); setConfig({ ...DEFAULTS, ...d }); setLogoPreview(d.logo_url || ""); }
    } catch { /* defaults */ }
  }, []);

  const handleLogo = (file: File) => {
    if (file.size > 200000) { setMsg("Logo too large (max 200KB)"); return; }
    const reader = new FileReader();
    reader.onload = (e) => { const url = e.target?.result as string; setLogoPreview(url); setConfig({ ...config, logo_url: url }); };
    reader.readAsDataURL(file);
  };

  const save = async () => {
    setSaving(true);
    try {
      await fetch(`${API_BASE}/api/v1/identity/branding/config`, {
        method: "PUT", headers: { "Content-Type": "application/json", ...authHeader() },
        body: JSON.stringify(config),
      });
    } catch { /* ok */ }
    setSaving(false);
    setMsg(t("brandingConfig.saved"));
    setTimeout(() => setMsg(null), 3000);
  };

  const reset = () => { setConfig(DEFAULTS); setLogoPreview(""); };

  const applyPreset = (preset: typeof PRESETS[0]) => {
    setConfig({ ...config, primary_color: preset.primary, secondary_color: preset.secondary, accent_color: preset.accent });
  };

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-950 p-4 md:p-8">
      <div className="max-w-4xl mx-auto">
        <div className="mb-6">
          <div className="flex items-center gap-3 mb-1">
            <Palette className="w-7 h-7 text-blue-600" />
            <h1 className="text-2xl font-bold text-gray-900 dark:text-white">{t("brandingConfig.title")}</h1>
          </div>
          <p className="text-sm text-gray-500 dark:text-gray-400">{t("brandingConfig.description")}</p>
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          {/* Config Form */}
          <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-6 space-y-5">
            {/* App Name */}
            <div>
              <label className="block text-sm font-medium text-gray-900 dark:text-white mb-1">{t("brandingConfig.appName")}</label>
              <p className="text-xs text-gray-400 mb-2">{t("brandingConfig.appNameDesc")}</p>
              <input type="text" value={config.app_name} onChange={(e) => setConfig({ ...config, app_name: e.target.value })} placeholder={t("brandingConfig.appNamePlaceholder")}
                className="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 text-sm text-gray-900 dark:text-white" />
            </div>

            {/* Logo Upload */}
            <div>
              <label className="block text-sm font-medium text-gray-900 dark:text-white mb-1">{t("brandingConfig.logo")}</label>
              <p className="text-xs text-gray-400 mb-2">{t("brandingConfig.logoDesc")}</p>
              <div onClick={() => fileRef.current?.click()}
                onDrop={(e) => { e.preventDefault(); const f = e.dataTransfer.files[0]; if (f) handleLogo(f); }}
                onDragOver={(e) => e.preventDefault()}
                className="border-2 border-dashed rounded-xl p-6 text-center cursor-pointer border-gray-300 dark:border-gray-700 hover:border-blue-400">
                {logoPreview ? (
                  <img src={logoPreview} alt="Logo" className="h-16 mx-auto object-contain" />
                ) : (
                  <>
                    <Upload className="w-8 h-8 mx-auto mb-2 text-gray-400" />
                    <p className="text-xs text-gray-500">{t("brandingConfig.logoUpload")}</p>
                  </>
                )}
                <input ref={fileRef} type="file" accept=".svg,.png,.jpg" onChange={(e) => { const f = e.target.files?.[0]; if (f) handleLogo(f); }} className="hidden" />
              </div>
            </div>

            {/* Color Presets */}
            <div>
              <label className="block text-sm font-medium text-gray-900 dark:text-white mb-2">{t("brandingConfig.presets")}</label>
              <div className="grid grid-cols-2 gap-2">
                {PRESETS.map((p) => (
                  <button key={p.name} onClick={() => applyPreset(p)} className="flex items-center gap-2 p-2 rounded-lg border border-gray-200 dark:border-gray-700 hover:border-gray-300 text-xs">
                    <div className="flex gap-0.5"><div className="w-4 h-4 rounded" style={{ backgroundColor: p.primary }} /><div className="w-4 h-4 rounded" style={{ backgroundColor: p.secondary }} /><div className="w-4 h-4 rounded" style={{ backgroundColor: p.accent }} /></div>
                    <span className="text-gray-700 dark:text-gray-300">{p.name}</span>
                  </button>
                ))}
              </div>
            </div>

            {/* Color Pickers */}
            <div>
              <label className="block text-sm font-medium text-gray-900 dark:text-white mb-1">{t("brandingConfig.colors")}</label>
              <p className="text-xs text-gray-400 mb-3">{t("brandingConfig.colorsDesc")}</p>
              <div className="space-y-2">
                <ColorRow label={t("brandingConfig.primaryColor")} value={config.primary_color} onChange={(v) => setConfig({ ...config, primary_color: v })} />
                <ColorRow label={t("brandingConfig.secondaryColor")} value={config.secondary_color} onChange={(v) => setConfig({ ...config, secondary_color: v })} />
                <ColorRow label={t("brandingConfig.accentColor")} value={config.accent_color} onChange={(v) => setConfig({ ...config, accent_color: v })} />
              </div>
            </div>

            {/* Login Subtitle */}
            <div>
              <label className="block text-sm font-medium text-gray-900 dark:text-white mb-1">{t("brandingConfig.loginSubtitle")}</label>
              <p className="text-xs text-gray-400 mb-2">{t("brandingConfig.loginSubtitleDesc")}</p>
              <input type="text" value={config.login_subtitle} onChange={(e) => setConfig({ ...config, login_subtitle: e.target.value })} placeholder={t("brandingConfig.loginSubtitlePlaceholder")}
                className="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 text-sm text-gray-900 dark:text-white" />
            </div>

            {/* Footer Text */}
            <div>
              <label className="block text-sm font-medium text-gray-900 dark:text-white mb-1">{t("brandingConfig.footerText")}</label>
              <p className="text-xs text-gray-400 mb-2">{t("brandingConfig.footerTextDesc")}</p>
              <input type="text" value={config.footer_text} onChange={(e) => setConfig({ ...config, footer_text: e.target.value })} placeholder={t("brandingConfig.footerPlaceholder")}
                className="w-full px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 text-sm text-gray-900 dark:text-white" />
            </div>

            {msg && (
              <div className="flex items-center gap-2 px-4 py-2 rounded-lg bg-green-50 text-green-700 dark:bg-green-950 dark:text-green-300 text-sm">
                <Check className="w-4 h-4" />{msg}
              </div>
            )}

            <div className="flex gap-2 pt-2">
              <button onClick={reset} className="flex items-center gap-1.5 px-4 py-2 bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-gray-400 rounded-lg text-sm font-medium">
                <RotateCcw className="w-4 h-4" />{t("brandingConfig.reset")}
              </button>
              <button onClick={save} disabled={saving}
                className="flex items-center gap-2 px-6 py-2 bg-blue-600 hover:bg-blue-700 disabled:opacity-50 text-white rounded-lg text-sm font-medium">
                {saving ? <Loader2 className="w-4 h-4 animate-spin" /> : <Save className="w-4 h-4" />}{t("brandingConfig.save")}
              </button>
            </div>
          </div>

          {/* Live Preview */}
          <div className="space-y-4">
            <div className="bg-white dark:bg-gray-900 rounded-xl border border-gray-200 dark:border-gray-800 p-4">
              <span className="text-xs font-semibold uppercase tracking-wider text-gray-400 mb-3 flex items-center gap-1"><Eye className="w-3 h-3" />{t("brandingConfig.preview")}</span>

              {/* Mini login preview */}
              <div className="rounded-xl border border-gray-200 dark:border-gray-700 overflow-hidden">
                <div className="p-6 flex flex-col items-center" style={{ background: `linear-gradient(135deg, ${config.primary_color}15, ${config.secondary_color}15)` }}>
                  <div className="w-12 h-12 rounded-xl flex items-center justify-center text-white font-bold shadow-lg mb-3"
                    style={{ background: `linear-gradient(135deg, ${config.primary_color}, ${config.secondary_color})` }}>
                    {logoPreview ? <img src={logoPreview} alt="" className="w-8 h-8 object-contain" /> : config.app_name[0]}
                  </div>
                  <h3 className="text-lg font-bold text-gray-900 dark:text-white">{config.app_name}</h3>
                  <p className="text-xs text-gray-500 mt-0.5">{config.login_subtitle}</p>
                  <button className="mt-3 px-6 py-2 text-white rounded-xl text-xs font-medium shadow" style={{ background: `linear-gradient(90deg, ${config.primary_color}, ${config.secondary_color})` }}>
                    Sign In
                  </button>
                </div>
                <div className="p-3 border-t border-gray-100 dark:border-gray-800 text-center">
                  <span className="text-xs text-gray-400">{config.footer_text}</span>
                </div>
              </div>

              {/* Sidebar preview */}
              <div className="mt-3 rounded-lg border border-gray-200 dark:border-gray-700 p-2">
                <div className="flex items-center gap-2 p-2 rounded-lg" style={{ backgroundColor: `${config.primary_color}15` }}>
                  <div className="w-6 h-6 rounded flex items-center justify-center text-white text-xs font-bold" style={{ background: config.primary_color }}>{config.app_name[0]}</div>
                  <span className="text-sm font-medium text-gray-900 dark:text-white">{config.app_name}</span>
                </div>
                <div className="mt-1 space-y-0.5">
                  {["Dashboard", "Users", "Settings"].map((item) => (
                    <div key={item} className="flex items-center gap-2 px-2 py-1.5 rounded text-xs text-gray-500">
                      <div className="w-3 h-3 rounded" style={{ background: item === "Dashboard" ? config.primary_color : "#d1d5db" }} />
                      {item}
                    </div>
                  ))}
                </div>
              </div>

              {/* Color swatches */}
              <div className="mt-3 flex items-center gap-3 justify-center">
                <div className="flex items-center gap-1"><div className="w-6 h-6 rounded shadow" style={{ background: config.primary_color }} /><span className="text-xs text-gray-400">Primary</span></div>
                <div className="flex items-center gap-1"><div className="w-6 h-6 rounded shadow" style={{ background: config.secondary_color }} /><span className="text-xs text-gray-400">Secondary</span></div>
                <div className="flex items-center gap-1"><div className="w-6 h-6 rounded shadow" style={{ background: config.accent_color }} /><span className="text-xs text-gray-400">Accent</span></div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

function ColorRow({ label, value, onChange }: { label: string; value: string; onChange: (v: string) => void }) {
  return (
    <div className="flex items-center gap-3">
      <input type="color" value={value} onChange={(e) => onChange(e.target.value)} className="w-10 h-10 rounded-lg border border-gray-200 dark:border-gray-700 cursor-pointer" />
      <span className="text-sm text-gray-700 dark:text-gray-300 flex-1">{label}</span>
      <input type="text" value={value} onChange={(e) => onChange(e.target.value)} className="w-20 px-2 py-1 rounded border border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 text-xs font-mono text-gray-900 dark:text-white" />
    </div>
  );
}
