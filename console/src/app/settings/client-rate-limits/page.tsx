"use client";
import { useTranslations } from "@/lib/i18n";

import { useState, useEffect, useCallback } from "react";
import { Gauge, Save, RotateCcw } from "lucide-react";

interface RateLimitConfig {
  client_id: string;
  client_name: string;
  requests_per_minute: number;
  burst: number;
  daily_quota: number;
  enabled: boolean;
}

interface Client {
  client_id: string;
  client_name: string;
}

export default function ClientRateLimitsPage() {
  const t = useTranslations();
  const [clients, setClients] = useState<Client[]>([]);
  const [selectedId, setSelectedId] = useState("");
  const [config, setConfig] = useState<RateLimitConfig | null>(null);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [saved, setSaved] = useState(false);

  const fetchClients = useCallback(async () => {
    try {
      const res = await fetch("/api/v1/oauth/clients", { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) { const data = await res.json(); setClients(data.clients || data || []); }
    } catch { /* noop */ }
  }, []);

  const fetchConfig = useCallback(async (id: string) => {
    setLoading(true);
    try {
      const res = await fetch(`/api/v1/oauth/client-rate-limits?client_id=${encodeURIComponent(id)}`, { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) setConfig(await res.json());
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchClients(); }, [fetchClients]);
  useEffect(() => { if (selectedId) fetchConfig(selectedId); }, [selectedId, fetchConfig]);

  const save = async () => {
    if (!config) return;
    setSaving(true);
    try { await fetch("/api/v1/oauth/client-rate-limits", { method: "PUT", headers: { "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" }, body: JSON.stringify(config) }); setSaved(true); setTimeout(() => setSaved(false), 2000); }
    catch { /* noop */ }
    finally { setSaving(false); }
  };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><Gauge className="w-6 h-6 text-cyan-500" /> {t("backend.clientRateLimits.title")}</h1>
        <p className="text-sm text-gray-500 mt-1">Configure per-client rate limiting: requests/min, burst, and daily quota.</p>
      </div>

      <select aria-label="Selected id" value={selectedId} onChange={(e) => setSelectedId(e.target.value)} className="px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm">
        <option value="">{t("backend.clientRateLimits.selectClient")}</option>
        {clients.map((c) => <option key={c.client_id} value={c.client_id}>{c.client_name}</option>)}
      </select>

      {config && (
        <div className="rounded-lg border dark:border-gray-800 p-6 space-y-4 max-w-lg">
          <div className="flex items-center justify-between">
            <div><span className="font-semibold">{config.client_name}</span><p className="text-xs text-gray-400 font-mono">{config.client_id}</p></div>
            <label className="flex items-center gap-2 text-sm"><input aria-label="Config" type="checkbox" checked={config.enabled} onChange={(e) => setConfig({ ...config, enabled: e.target.checked })} className="rounded" /> Enabled</label>
          </div>

          <div className="grid grid-cols-3 gap-4">
            <div><label className="text-sm font-medium">Requests / min</label><input aria-label="config" type="number" min={0} value={config.requests_per_minute} onChange={(e) => setConfig({ ...config, requests_per_minute: parseInt(e.target.value) || 0 })} className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm" /></div>
            <div><label className="text-sm font-medium">{t("backend.clientRateLimits.burst")}</label><input aria-label="config" type="number" min={0} value={config.burst} onChange={(e) => setConfig({ ...config, burst: parseInt(e.target.value) || 0 })} className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm" /></div>
            <div><label className="text-sm font-medium">{t("backend.clientRateLimits.dailyQuota")}</label><input aria-label="config" type="number" min={0} value={config.daily_quota} onChange={(e) => setConfig({ ...config, daily_quota: parseInt(e.target.value) || 0 })} className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm" /></div>
          </div>

          <div className="flex items-center gap-2">
            <button aria-label="Save" onClick={save} disabled={saving} className="px-4 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium hover:bg-blue-700 disabled:opacity-50 flex items-center gap-2"><Save className="w-4 h-4" /> {saving ? "Saving..." : "Save"}</button>
            <button onClick={() => fetchConfig(selectedId)} className="px-4 py-2 rounded-lg border dark:border-gray-700 text-sm flex items-center gap-2"><RotateCcw className="w-4 h-4" /> Reset</button>
            {saved && <span className="text-sm text-green-600">Saved!</span>}
          </div>
        </div>
      )}
      {!config && !loading && selectedId && <p className="text-sm text-gray-500">Loading...</p>}
    </div>
  );
}
