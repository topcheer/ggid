"use client";

import React, { useState, useEffect } from "react";
import { useApi } from "@/lib/api";
import { Eye, Loader2, AlertCircle, X, Save, Monitor } from "lucide-react";

interface ConsentConfig {
  logo_url: string;
  primary_color: string;
  app_name: string;
  privacy_url: string;
  terms_url: string;
  support_email: string;
  custom_message: string;
  show_scopes: boolean;
  show_permissions: boolean;
  require_explicit_consent: boolean;
}

const defaultConfig: ConsentConfig = {
  logo_url: "",
  primary_color: "#4f46e5",
  app_name: "My App",
  privacy_url: "",
  terms_url: "",
  support_email: "",
  custom_message: "",
  show_scopes: true,
  show_permissions: true,
  require_explicit_consent: false,
};

export default function ConsentScreenPage() {
  const { apiFetch } = useApi();
  const [config, setConfig] = useState<ConsentConfig | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    const fetchData = async () => {
      setLoading(true);
      setError(null);
      try {
        const data = await apiFetch<ConsentConfig>("/api/v1/oauth/consent-screen/config");
        setConfig(data || defaultConfig);
      } catch (e) {
        setConfig(defaultConfig);
        setError(e instanceof Error ? e.message : "Failed to load config");
      } finally {
        setLoading(false);
      }
    };
    fetchData();
  }, [apiFetch]);

  const handleSave = async () => {
    if (!config) return;
    setSaving(true);
    setError(null);
    try {
      await apiFetch("/api/v1/oauth/consent-screen/config", { method: "PUT", body: JSON.stringify(config) });
    } catch (e) {
      setError(e instanceof Error ? e.message : "Save failed");
    } finally {
      setSaving(false);
    }
  };

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";
  const inputCls = "w-full rounded-lg border border-gray-300 px-3 py-2 text-sm dark:border-gray-600 dark:bg-gray-900 dark:text-gray-200";

  return (
    <div className="space-y-6">
      <div>
        <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><Eye className="h-6 w-6 text-indigo-600" /> Consent Screen</h1>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">Customize the OAuth consent page with live preview.</p>
      </div>

      {error && <div className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400"><AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} aria-label="Dismiss error" className="ml-auto"><X className="h-4 w-4" /></button></div>}

      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-indigo-600" /></div>
      : config ? (
        <div className="grid grid-cols-2 gap-6">
          {/* Edit form */}
          <div className={cardCls}>
            <h3 className="mb-4 text-sm font-semibold text-gray-700 dark:text-gray-300">Configuration</h3>
            <div className="space-y-4">
              <div><label className="mb-1 block text-xs font-semibold uppercase text-gray-400">App Name</label><input value={config.app_name} onChange={(e) => setConfig({ ...config, app_name: e.target.value })} aria-label="Application name" className={inputCls} /></div>
              <div><label className="mb-1 block text-xs font-semibold uppercase text-gray-400">Logo URL</label><input value={config.logo_url} onChange={(e) => setConfig({ ...config, logo_url: e.target.value })} placeholder="https://..." aria-label="Logo URL" className={inputCls} /></div>
              <div className="flex gap-4">
                <div className="flex-1"><label className="mb-1 block text-xs font-semibold uppercase text-gray-400">Primary Color</label><div className="flex items-center gap-2"><input type="color" value={config.primary_color} onChange={(e) => setConfig({ ...config, primary_color: e.target.value })} aria-label="Primary color" className="h-9 w-12 rounded border border-gray-300 dark:border-gray-600" /><input value={config.primary_color} onChange={(e) => setConfig({ ...config, primary_color: e.target.value })} aria-label="Primary color hex" className={`${inputCls} font-mono`} /></div></div>
                <div className="flex-1"><label className="mb-1 block text-xs font-semibold uppercase text-gray-400">Support Email</label><input value={config.support_email} onChange={(e) => setConfig({ ...config, support_email: e.target.value })} aria-label="Support email" className={inputCls} /></div>
              </div>
              <div><label className="mb-1 block text-xs font-semibold uppercase text-gray-400">Privacy URL</label><input value={config.privacy_url} onChange={(e) => setConfig({ ...config, privacy_url: e.target.value })} aria-label="Privacy URL" className={inputCls} /></div>
              <div><label className="mb-1 block text-xs font-semibold uppercase text-gray-400">Terms URL</label><input value={config.terms_url} onChange={(e) => setConfig({ ...config, terms_url: e.target.value })} aria-label="Terms URL" className={inputCls} /></div>
              <div><label className="mb-1 block text-xs font-semibold uppercase text-gray-400">Custom Message</label><textarea value={config.custom_message} onChange={(e) => setConfig({ ...config, custom_message: e.target.value })} aria-label="Custom message" rows={2} className={inputCls} /></div>
              <div className="flex flex-wrap gap-4">
                <label className="flex items-center gap-2 text-sm text-gray-600 dark:text-gray-300"><input type="checkbox" checked={config.show_scopes} onChange={(e) => setConfig({ ...config, show_scopes: e.target.checked })} aria-label="Show scopes" />Show scopes</label>
                <label className="flex items-center gap-2 text-sm text-gray-600 dark:text-gray-300"><input type="checkbox" checked={config.show_permissions} onChange={(e) => setConfig({ ...config, show_permissions: e.target.checked })} aria-label="Show permissions" />Show permissions</label>
                <label className="flex items-center gap-2 text-sm text-gray-600 dark:text-gray-300"><input type="checkbox" checked={config.require_explicit_consent} onChange={(e) => setConfig({ ...config, require_explicit_consent: e.target.checked })} aria-label="Require explicit consent" />Require explicit consent</label>
              </div>
              <button onClick={handleSave} disabled={saving} aria-label="Save consent configuration" className="flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50">{saving ? <Loader2 className="h-4 w-4 animate-spin" /> : <Save className="h-4 w-4" />}Save</button>
            </div>
          </div>

          {/* Live preview */}
          <div className={cardCls}>
            <h3 className="mb-4 flex items-center gap-2 text-sm font-semibold text-gray-700 dark:text-gray-300"><Monitor className="h-4 w-4" /> Live Preview</h3>
            <div className="overflow-hidden rounded-lg border border-gray-200 dark:border-gray-700">
              <div className="flex items-center gap-1.5 border-b border-gray-200 bg-gray-50 px-3 py-2 dark:border-gray-700 dark:bg-gray-900"><div className="h-2.5 w-2.5 rounded-full bg-red-400" /><div className="h-2.5 w-2.5 rounded-full bg-yellow-400" /><div className="h-2.5 w-2.5 rounded-full bg-green-400" /></div>
              <div className="p-6" style={{ backgroundColor: "#f9fafb" }}>
                <div className="mx-auto max-w-sm rounded-xl bg-white p-6 shadow-lg">
                  {config.logo_url && <img src={config.logo_url} alt={`${config.app_name} logo`} className="mx-auto mb-4 h-12 w-12 rounded" onError={(e) => { (e.target as HTMLImageElement).style.display = "none"; }} />}
                  <h2 className="text-center text-lg font-bold text-gray-900" style={{ color: config.primary_color }}>{config.app_name}</h2>
                  {config.custom_message && <p className="mt-2 text-center text-sm text-gray-500">{config.custom_message}</p>}
                  {config.show_scopes && (<div className="mt-4"><div className="mb-2 text-xs font-semibold uppercase text-gray-400">Requested scopes</div><div className="space-y-1"><div className="flex items-center gap-2 text-sm text-gray-600"><div className="h-2 w-2 rounded-full" style={{ backgroundColor: config.primary_color }} />profile, email, openid</div></div></div>)}
                  {config.show_permissions && (<div className="mt-3"><div className="mb-2 text-xs font-semibold uppercase text-gray-400">Permissions</div><div className="text-sm text-gray-500">Read and write your profile data</div></div>)}
                  <div className="mt-6 flex gap-3">
                    <button aria-label="Preview allow" className="flex-1 rounded-lg py-2 text-sm font-medium text-white" style={{ backgroundColor: config.primary_color }}>Allow</button>
                    <button aria-label="Preview deny" className="flex-1 rounded-lg border border-gray-300 py-2 text-sm font-medium text-gray-600 dark:border-gray-600">Deny</button>
                  </div>
                  {config.privacy_url && <a href={config.privacy_url} className="mt-3 block text-center text-xs text-gray-400 hover:underline">Privacy Policy</a>}
                </div>
              </div>
            </div>
          </div>
        </div>
      ) : null}
    </div>
  );
}
