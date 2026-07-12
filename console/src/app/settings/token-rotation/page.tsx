"use client";

import { useState, useEffect, useCallback } from "react";
import { RefreshCw, Save, ToggleLeft, ToggleRight } from "lucide-react";

interface RotationConfig {
  client_id: string;
  client_name: string;
  enabled: boolean;
  interval_days: number;
  max_age_hours: number;
  notify_before_hours: number;
}

interface Client { client_id: string; client_name: string; }

export default function TokenRotationPage() {
  const [clients, setClients] = useState<Client[]>([]);
  const [selectedId, setSelectedId] = useState("");
  const [config, setConfig] = useState<RotationConfig | null>(null);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [saved, setSaved] = useState(false);

  const fetchClients = useCallback(async () => {
    try {
      const res = await fetch("/api/v1/oauth/clients", { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) { const d = await res.json(); setClients(d.clients || d || []); }
    } catch { /* noop */ }
  }, []);

  const fetchConfig = useCallback(async (id: string) => {
    setLoading(true);
    try {
      const res = await fetch(`/api/v1/oauth/token-rotation?client_id=${encodeURIComponent(id)}`, { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) setConfig(await res.json());
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchClients(); }, [fetchClients]);
  useEffect(() => { if (selectedId) fetchConfig(selectedId); }, [selectedId, fetchConfig]);

  const save = async () => {
    if (!config) return;
    setSaving(true);
    try { await fetch("/api/v1/oauth/token-rotation", { method: "PUT", headers: { "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" }, body: JSON.stringify(config) }); setSaved(true); setTimeout(() => setSaved(false), 2000); }
    catch { /* noop */ }
    finally { setSaving(false); }
  };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><RefreshCw className="w-6 h-6 text-teal-500" /> Token Rotation</h1>
        <p className="text-sm text-gray-500 mt-1">Configure automatic token rotation policies per OAuth client.</p>
      </div>

      <select value={selectedId} onChange={(e) => setSelectedId(e.target.value)} className="px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm">
        <option value="">Select Client</option>
        {clients.map((c) => <option key={c.client_id} value={c.client_id}>{c.client_name}</option>)}
      </select>

      {config && (
        <div className="rounded-lg border dark:border-gray-800 p-6 space-y-5 max-w-lg">
          <div className="flex items-center justify-between">
            <div><span className="font-semibold">{config.client_name}</span><p className="text-xs text-gray-400 font-mono">{config.client_id}</p></div>
            <button onClick={() => setConfig({ ...config, enabled: !config.enabled })} className="flex items-center gap-1 text-sm">
              {config.enabled ? <ToggleRight className="w-8 h-8 text-green-500" /> : <ToggleLeft className="w-8 h-8 text-gray-400" />}
              <span className={config.enabled ? "text-green-600" : "text-gray-500"}>{config.enabled ? "Enabled" : "Disabled"}</span>
            </button>
          </div>

          <div>
            <label className="text-sm font-medium">Rotation Interval: {config.interval_days} days</label>
            <input type="range" min={1} max={90} value={config.interval_days} onChange={(e) => setConfig({ ...config, interval_days: parseInt(e.target.value) })} className="w-full mt-2 accent-teal-500" />
            <div className="flex justify-between text-xs text-gray-400 mt-1"><span>1d</span><span>30d</span><span>90d</span></div>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div><label className="text-sm font-medium">Max Age (hours)</label><input type="number" min={1} value={config.max_age_hours} onChange={(e) => setConfig({ ...config, max_age_hours: parseInt(e.target.value) || 0 })} className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm" /></div>
            <div><label className="text-sm font-medium">Notify Before (hours)</label><input type="number" min={1} value={config.notify_before_hours} onChange={(e) => setConfig({ ...config, notify_before_hours: parseInt(e.target.value) || 0 })} className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm" /></div>
          </div>

          <div className="flex items-center gap-2">
            <button onClick={save} disabled={saving} className="px-4 py-2 rounded-lg bg-teal-600 text-white text-sm font-medium hover:bg-teal-700 disabled:opacity-50 flex items-center gap-2"><Save className="w-4 h-4" /> {saving ? "Saving..." : "Save"}</button>
            {saved && <span className="text-sm text-green-600">Saved!</span>}
          </div>
        </div>
      )}
      {!config && selectedId && loading && <p className="text-sm text-gray-500">Loading...</p>}
    </div>
  );
}
