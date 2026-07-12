"use client";
import { useState, useEffect, useCallback } from "react";
import { Eye, Save, Play } from "lucide-react";

interface ConsentConfig { logo_url: string; privacy_policy_url: string; tos_url: string; show_skip_consent: boolean; remember_consent_duration_days: number; scope_descriptions: Record<string, string>; pre_approved_apps: { client_id: string; client_name: string }[]; }

export default function OauthConsentFlowPage() {
  const [config, setConfig] = useState<ConsentConfig | null>(null);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [saved, setSaved] = useState(false);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/v1/oauth/consent-flow-config", { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) setConfig(await res.json());
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  const save = async () => {
    if (!config) return;
    setSaving(true);
    try { await fetch("/api/v1/oauth/consent-flow-config", { method: "PUT", headers: { "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" }, body: JSON.stringify(config) }); setSaved(true); setTimeout(() => setSaved(false), 2000); }
    catch { /* noop */ }
    finally { setSaving(false); }
  };

  if (!config) return <p className="text-sm text-gray-500 text-center py-8">Loading...</p>;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div><h1 className="text-2xl font-bold flex items-center gap-2"><Eye className="w-6 h-6 text-blue-500" /> OAuth Consent Flow</h1><p className="text-sm text-gray-500 mt-1">Configure the consent screen, scope descriptions, and pre-approved apps.</p></div>
        <div className="flex gap-2"><button onClick={save} disabled={saving} className="px-4 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium disabled:opacity-50 flex items-center gap-2"><Save className="w-4 h-4" /> {saving ? "Saving..." : "Save"}</button>{saved && <span className="text-sm text-green-600 self-center">Saved!</span>}</div>
      </div>

      <div className="rounded-lg border dark:border-gray-800 p-4"><h3 className="text-sm font-semibold mb-3">Consent Screen Preview</h3><div className="rounded-lg border dark:border-gray-700 p-6 max-w-sm"><div className="flex items-center gap-3 mb-4">{config.logo_url && <img src={config.logo_url} alt="logo" className="w-10 h-10 rounded" />}<div><h4 className="font-semibold">MyApp</h4><p className="text-xs text-gray-500">is requesting access to your account</p></div></div><div className="space-y-2 mb-4">{Object.entries(config.scope_descriptions).slice(0, 3).map(([scope, desc]) => (<div key={scope} className="text-sm flex items-start gap-2"><span className="w-2 h-2 rounded-full bg-blue-500 mt-1.5" /><div><span className="font-medium">{scope}</span><p className="text-xs text-gray-500">{desc}</p></div></div>))}</div><button className="w-full py-2 rounded-lg bg-blue-600 text-white text-sm font-medium">Allow</button><button className="w-full py-2 rounded-lg text-sm text-gray-500 mt-1">Deny</button></div></div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div className="rounded-lg border dark:border-gray-800 p-4 space-y-3"><h3 className="text-sm font-semibold">Branding</h3><div><label className="text-xs font-medium text-gray-500">Logo URL</label><input type="text" value={config.logo_url} onChange={(e) => setConfig({ ...config, logo_url: e.target.value })} className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm" /></div><div><label className="text-xs font-medium text-gray-500">Privacy Policy URL</label><input type="text" value={config.privacy_policy_url} onChange={(e) => setConfig({ ...config, privacy_policy_url: e.target.value })} className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm" /></div><div><label className="text-xs font-medium text-gray-500">Terms of Service URL</label><input type="text" value={config.tos_url} onChange={(e) => setConfig({ ...config, tos_url: e.target.value })} className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm" /></div></div>
        <div className="rounded-lg border dark:border-gray-800 p-4 space-y-3"><h3 className="text-sm font-semibold">Settings</h3><label className="flex items-center gap-2 text-sm"><input type="checkbox" checked={config.show_skip_consent} onChange={(e) => setConfig({ ...config, show_skip_consent: e.target.checked })} className="rounded" /> Show skip consent for trusted clients</label><div><label className="text-xs font-medium text-gray-500">Remember consent (days)</label><input type="number" min={0} value={config.remember_consent_duration_days} onChange={(e) => setConfig({ ...config, remember_consent_duration_days: parseInt(e.target.value) || 0 })} className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm" /></div><button className="px-4 py-2 rounded-lg border dark:border-gray-700 text-sm flex items-center gap-2"><Play className="w-4 h-4" /> Test Flow</button></div>
      </div>

      <div className="rounded-lg border dark:border-gray-800 p-4"><h3 className="text-sm font-semibold mb-3">Scope Descriptions</h3><div className="space-y-2">{Object.entries(config.scope_descriptions).map(([scope, desc]) => (<div key={scope} className="flex items-center gap-2"><span className="font-mono text-xs w-32">{scope}</span><input type="text" value={desc} onChange={(e) => { const next = { ...config.scope_descriptions }; next[scope] = e.target.value; setConfig({ ...config, scope_descriptions: next }); }} className="flex-1 px-3 py-1.5 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm" /></div>))}</div></div>

      {config.pre_approved_apps.length > 0 && (<div className="rounded-lg border dark:border-gray-800 p-4"><h3 className="text-sm font-semibold mb-2">Pre-Approved Apps</h3><div className="space-y-1">{config.pre_approved_apps.map((a) => (<div key={a.client_id} className="flex items-center justify-between text-sm py-1"><span className="font-medium">{a.client_name}</span><span className="text-xs text-gray-400 font-mono">{a.client_id}</span></div>))}</div></div>)}
    </div>
  );
}
