"use client";

import { useState, useCallback, useEffect } from "react";
import { Clock, Save, Shield, Loader2, AlertCircle } from "lucide-react";

interface RoleOverride {
  role: string;
  idle_timeout_minutes: number;
  absolute_timeout_hours: number;
}

interface Config {
  idle_timeout_minutes: number;
  absolute_timeout_hours: number;
  warning_before_minutes: number;
  grace_period: boolean;
  enforce_on_mobile: boolean;
  role_overrides: RoleOverride[];
}

export default function SessionTimeoutConfigPage() {
  const [config, setConfig] = useState<Config>({ idle_timeout_minutes: 30, absolute_timeout_hours: 8, warning_before_minutes: 5, grace_period: true, enforce_on_mobile: false, role_overrides: [{ role: "admin", idle_timeout_minutes: 15, absolute_timeout_hours: 4 }, { role: "viewer", idle_timeout_minutes: 60, absolute_timeout_hours: 12 }] });
  const [saving, setSaving] = useState(false);
  const [saved, setSaved] = useState(false);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  useEffect(() => {
    setLoading(true); setError("");
    fetch("/api/v1/auth/session-timeout-config", { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } })
      .then(async (res) => { if (res.ok) { const data = await res.json(); if (data) setConfig(data); } })
      .catch(() => { /* use defaults */ })
      .finally(() => setLoading(false));
  }, []);

  const save = useCallback(async () => {
    setSaving(true); setError("");
    try {
      const res = await fetch("/api/v1/auth/session-timeout-config", { method: "PUT", headers: { "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" }, body: JSON.stringify(config) });
      if (!res.ok) throw new Error(`Save failed: HTTP ${res.status}`);
      setSaved(true); setTimeout(() => setSaved(false), 2000);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to save configuration");
    } finally { setSaving(false); }
  }, [config]);

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div><h1 className="text-2xl font-bold flex items-center gap-2"><Clock className="w-6 h-6 text-blue-500" /> Session Timeout Config</h1><p className="text-sm text-gray-500 mt-1">Configure idle and absolute session timeouts with per-role overrides.</p></div>
        <button onClick={save} disabled={saving} className="px-4 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium hover:bg-blue-700 disabled:opacity-50 flex items-center gap-2"><Save className="w-4 h-4" /> {saving ? "Saving..." : "Save"}</button>
      </div>
      {saved && <div className="text-sm text-green-600">Saved!</div>}
      {error && <div className="rounded-lg border border-red-200 dark:border-red-900 bg-red-50 dark:bg-red-900/20 p-3 text-sm text-red-600 flex items-center gap-2"><AlertCircle className="w-4 h-4" /> {error}</div>}
      {loading && <div className="flex items-center gap-2 text-sm text-gray-500"><Loader2 className="w-4 h-4 animate-spin" /> Loading configuration...</div>}

      <div className="rounded-lg border dark:border-gray-800 p-6 space-y-4 max-w-lg">
        <div><label className="text-sm font-medium">Idle Timeout (minutes)</label><div className="flex items-center gap-3 mt-1"><input type="range" min={5} max={120} value={config.idle_timeout_minutes} onChange={(e) => setConfig({ ...config, idle_timeout_minutes: parseInt(e.target.value) })} className="flex-1" /><span className="text-sm font-bold w-12 text-right">{config.idle_timeout_minutes}m</span></div></div>
        <div><label className="text-sm font-medium">Absolute Timeout (hours)</label><div className="flex items-center gap-3 mt-1"><input type="range" min={1} max={24} value={config.absolute_timeout_hours} onChange={(e) => setConfig({ ...config, absolute_timeout_hours: parseInt(e.target.value) })} className="flex-1" /><span className="text-sm font-bold w-12 text-right">{config.absolute_timeout_hours}h</span></div></div>
        <div><label className="text-sm font-medium">Warning Before (minutes)</label><input type="number" min={1} max={30} value={config.warning_before_minutes} onChange={(e) => setConfig({ ...config, warning_before_minutes: parseInt(e.target.value) })} className="w-20 mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm" /></div>
        <div className="flex items-center gap-4"><label className="flex items-center gap-2 cursor-pointer"><input type="checkbox" checked={config.grace_period} onChange={(e) => setConfig({ ...config, grace_period: e.target.checked })} className="rounded" /><span className="text-sm">Grace Period</span></label><label className="flex items-center gap-2 cursor-pointer"><input type="checkbox" checked={config.enforce_on_mobile} onChange={(e) => setConfig({ ...config, enforce_on_mobile: e.target.checked })} className="rounded" /><span className="text-sm">Enforce on Mobile</span></label></div>
      </div>

      <div className="rounded-lg border dark:border-gray-800 p-4">
        <h3 className="text-sm font-semibold flex items-center gap-2 mb-3"><Shield className="w-4 h-4 text-gray-400" /> Per-Role Overrides</h3>
        <table className="w-full text-sm"><thead><tr><th className="px-4 py-2 text-left font-medium">Role</th><th className="px-4 py-2 text-left font-medium">Idle (min)</th><th className="px-4 py-2 text-left font-medium">Absolute (hrs)</th></tr></thead>
          <tbody className="divide-y dark:divide-gray-800">{config.role_overrides.map((r, i) => (<tr key={i}><td className="px-4 py-2 font-mono text-xs">{r.role}</td><td className="px-4 py-2"><input type="number" value={r.idle_timeout_minutes} onChange={(e) => { const o = [...config.role_overrides]; o[i] = { ...r, idle_timeout_minutes: parseInt(e.target.value) }; setConfig({ ...config, role_overrides: o }); }} className="w-16 px-2 py-1 rounded border dark:border-gray-700 dark:bg-gray-900 text-sm" /></td><td className="px-4 py-2"><input type="number" value={r.absolute_timeout_hours} onChange={(e) => { const o = [...config.role_overrides]; o[i] = { ...r, absolute_timeout_hours: parseInt(e.target.value) }; setConfig({ ...config, role_overrides: o }); }} className="w-16 px-2 py-1 rounded border dark:border-gray-700 dark:bg-gray-900 text-sm" /></td></tr>))}</tbody>
        </table>
      </div>
    </div>
  );
}
