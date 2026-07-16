"use client";
import { useState, useEffect, useCallback } from "react";
import { useTranslations } from "@/lib/i18n";
import { Palette, Upload, Eye, Save } from "lucide-react";
interface Branding { logo_url: string; primary_color: string; secondary_color: string; accent_color: string; custom_css: string; theme: "light" | "dark" | "auto"; custom_domain: string; }
export default function TenantBrandingPage() {
  const [branding, setBranding] = useState<Branding>({ logo_url: "", primary_color: "#3b82f6", secondary_color: "#1e40af", accent_color: "#f59e0b", custom_css: "", theme: "auto", custom_domain: "" });
  const [saving, setSaving] = useState(false);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showPreview, setShowPreview] = useState(false);
  const [saveMsg, setSaveMsg] = useState<{ type: "success" | "error"; text: string } | null>(null);
  const t = useTranslations();

  const loadBranding = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch("/api/v1/admin/branding", { headers: { "Authorization": `Bearer ${localStorage.getItem("ggid_access_token") || ""}`, "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (!res.ok) return null;
      const data = await res.json();
      if (data) setBranding(prev => ({ ...prev, ...data }));
    } catch (err) { setError(err instanceof Error ? err.message : "An error occurred"); }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { loadBranding(); }, [loadBranding]);

  const save = async () => {
    setSaving(true);
    setSaveMsg(null);
    try {
      const res = await fetch("/api/v1/admin/branding", { method: "PUT", headers: { "Authorization": `Bearer ${localStorage.getItem("ggid_access_token") || ""}`, "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" }, body: JSON.stringify(branding) });
      if (!res.ok) return null;
      setSaveMsg({ type: "success", text: t("tenantBranding.saved") });
    } catch (e) {
      setSaveMsg({ type: "error", text: e instanceof Error ? e.message : t("tenantBranding.saveFailed") });
    } finally { setSaving(false); }
  };

  if (loading) return (
    <div className="p-8 flex items-center justify-center">
      <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-purple-600" />
    </div>
  );

  if (error) return (
    <div className="p-8">
      <div className="rounded-lg border border-red-300 bg-red-50 dark:bg-red-950 dark:border-red-800 p-4">
        <p className="text-red-700 dark:text-red-400 text-sm font-medium">Error: {error}</p>
        <button aria-label="action" onClick={loadBranding} className="mt-2 px-4 py-1.5 rounded-lg bg-red-600 text-white text-sm hover:bg-red-700">{t("tenantBranding.retry")}</button>
      </div>
    </div>
  );

  return (
    <div className="space-y-6">
      {saveMsg && <div className={`rounded-lg border p-3 text-sm ${saveMsg.type === "success" ? "border-green-200 dark:border-green-900 bg-green-50 dark:bg-green-900/20 text-green-700 dark:text-green-400" : "border-red-200 dark:border-red-900 bg-red-50 dark:bg-red-900/20 text-red-700 dark:text-red-400"}`}>{saveMsg.text}</div>}
      <div className="flex items-center justify-between"><div><h1 className="text-2xl font-bold flex items-center gap-2"><Palette className="w-6 h-6 text-purple-500" /> Tenant Branding</h1><p className="text-sm text-gray-500 mt-1">{t("tenantBranding.subtitle")}</p></div><button onClick={save} disabled={saving} aria-label={t("tenantBranding.saveBranding")} className="px-4 py-2 rounded-lg bg-purple-600 text-white text-sm font-medium flex items-center gap-2"><Save className="w-4 h-4" /> {saving ? t("tenantBranding.saving") : t("tenantBranding.save")}</button></div>
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div className="space-y-4">
          <div className="rounded-lg border dark:border-gray-800 p-4"><label className="text-sm font-medium">{t("tenantBranding.logo")}</label><div className="mt-2 flex items-center gap-3"><div className="w-20 h-20 rounded-lg border-2 border-dashed dark:border-gray-700 flex items-center justify-center overflow-hidden">{branding.logo_url ? <img src={branding.logo_url} alt={t("tenantBranding.logoPreview")} className="w-full h-full object-contain" /> : <Upload className="w-6 h-6 text-gray-400" />}</div><input type="text" value={branding.logo_url} onChange={(e) => setBranding({ ...branding, logo_url: e.target.value })} placeholder="https://.../logo.png" aria-label={t("tenantBranding.logoUrl")} className="flex-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm font-mono" /></div></div>
          <div className="rounded-lg border dark:border-gray-800 p-4"><h3 className="text-sm font-semibold mb-3">{t("tenantBranding.colors")}</h3><div className="grid grid-cols-3 gap-3"><div><label className="text-xs text-gray-500">{t("tenantBranding.primary")}</label><div className="flex items-center gap-2 mt-1"><input type="color" value={branding.primary_color} onChange={(e) => setBranding({ ...branding, primary_color: e.target.value })} aria-label={t("tenantBranding.primaryColor")} className="w-8 h-8 rounded" /><input type="text" value={branding.primary_color} onChange={(e) => setBranding({ ...branding, primary_color: e.target.value })} aria-label="Primary color hex" className="w-20 px-2 py-1 rounded border dark:border-gray-700 dark:bg-gray-900 text-xs font-mono" /></div></div><div><label className="text-xs text-gray-500">{t("tenantBranding.secondary")}</label><div className="flex items-center gap-2 mt-1"><input type="color" value={branding.secondary_color} onChange={(e) => setBranding({ ...branding, secondary_color: e.target.value })} aria-label="Secondary color" className="w-8 h-8 rounded" /><input type="text" value={branding.secondary_color} onChange={(e) => setBranding({ ...branding, secondary_color: e.target.value })} aria-label="Secondary color hex" className="w-20 px-2 py-1 rounded border dark:border-gray-700 dark:bg-gray-900 text-xs font-mono" /></div></div><div><label className="text-xs text-gray-500">Accent</label><div className="flex items-center gap-2 mt-1"><input type="color" value={branding.accent_color} onChange={(e) => setBranding({ ...branding, accent_color: e.target.value })} aria-label="Accent color" className="w-8 h-8 rounded" /><input type="text" value={branding.accent_color} onChange={(e) => setBranding({ ...branding, accent_color: e.target.value })} aria-label="Accent color hex" className="w-20 px-2 py-1 rounded border dark:border-gray-700 dark:bg-gray-900 text-xs font-mono" /></div></div></div></div>
          <div className="rounded-lg border dark:border-gray-800 p-4"><label className="text-sm font-medium">Theme</label><select value={branding.theme} onChange={(e) => setBranding({ ...branding, theme: e.target.value as Branding["theme"] })} aria-label="Theme" className="ml-3 px-3 py-1.5 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm"><option value="light">Light</option><option value="dark">Dark</option><option value="auto">Auto</option></select></div>
          <div className="rounded-lg border dark:border-gray-800 p-4"><label className="text-sm font-medium">Custom Domain</label><input type="text" value={branding.custom_domain} onChange={(e) => setBranding({ ...branding, custom_domain: e.target.value })} placeholder="login.example.com" aria-label="Custom domain" className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm font-mono" /></div>
          <div className="rounded-lg border dark:border-gray-800 p-4"><label className="text-sm font-medium">Custom CSS</label><textarea value={branding.custom_css} onChange={(e) => setBranding({ ...branding, custom_css: e.target.value })} rows={6} placeholder="/* .login-btn { border-radius: 8px; } */" aria-label="Custom CSS" className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-xs font-mono" /></div>
        </div>
        <div><div className="flex items-center justify-between mb-2"><h3 className="text-sm font-semibold">Login Page Preview</h3><button onClick={() => setShowPreview(!showPreview)} aria-label="Toggle preview" className="text-xs flex items-center gap-1 text-gray-500"><Eye className="w-3.5 h-3.5" /> Toggle</button></div>{showPreview && <div className="rounded-lg border dark:border-gray-800 p-8" style={{ background: branding.theme === "dark" ? "#1a1a1a" : "#f8fafc" }}><div className="max-w-sm mx-auto rounded-lg shadow-lg p-6" style={{ background: branding.theme === "dark" ? "#222" : "#fff" }}>{branding.logo_url && <img src={branding.logo_url} alt="Branded logo" className="h-12 mx-auto mb-4" />}<h2 className="text-center text-lg font-bold mb-4" style={{ color: branding.primary_color }}>Sign In</h2><input autoComplete="current-password" type="text" placeholder="Email" aria-label="Preview email" className="w-full px-3 py-2 rounded-lg border dark:border-gray-700 mb-2 text-sm" style={{ borderColor: branding.secondary_color }} /><input type="password" placeholder="Password" aria-label="Preview password" className="w-full px-3 py-2 rounded-lg border dark:border-gray-700 mb-4 text-sm" style={{ borderColor: branding.secondary_color }} /><button className="w-full py-2 rounded-lg text-white text-sm font-medium" style={{ background: branding.primary_color }}>Sign In</button><style>{branding.custom_css}</style></div></div>}</div>
      </div>
    </div>
  );
}
