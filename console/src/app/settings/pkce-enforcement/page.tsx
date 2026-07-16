"use client";

import { useState, useEffect, useCallback } from "react";
import { Shield, Save } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface ClientOverride {
  client_id: string;
  client_name: string;
  required: boolean;
  challenge_method: "S256" | "plain";
}

interface Config {
  global_require_pkce: boolean;
  per_client: ClientOverride[];
  exempted_clients: string[];
  compliance_pct: number;
}

export default function PkceEnforcementPage() {
  const t = useTranslations();
  const [config, setConfig] = useState<Config | null>(null);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try { const res = await fetch("/api/v1/oauth/pkce-enforcement", { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } }); if (res.ok) setConfig(await res.json()); }
    catch { /* noop */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  const save = async () => {
    if (!config) return;
    setSaving(true);
    try { await fetch("/api/v1/oauth/pkce-enforcement", { method: "PUT", headers: { "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" }, body: JSON.stringify(config) }); }
    catch { /* noop */ }
    finally { setSaving(false); }
  };

  const gaugeColor = config ? (config.compliance_pct >= 90 ? "#10b981" : config.compliance_pct >= 70 ? "#f59e0b" : "#ef4444") : "#3b82f6";

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div><h1 className="text-2xl font-bold flex items-center gap-2"><Shield className="w-6 h-6 text-green-500" /> PKCE Enforcement</h1><p className="text-sm text-gray-500 mt-1">Enforce PKCE globally or per-client with challenge method configuration.</p></div>
        {config && <button onClick={save} disabled={saving} className="px-4 py-2 rounded-lg bg-green-600 text-white text-sm font-medium hover:bg-green-700 disabled:opacity-50 flex items-center gap-2"><Save className="w-4 h-4" /> Save</button>}
      </div>

      {config && (
        <>
          <div className="flex items-center gap-6">
            <label className="flex items-center gap-3 cursor-pointer"><input type="checkbox" checked={config.global_require_pkce} onChange={(e) => setConfig({ ...config, global_require_pkce: e.target.checked })} className="rounded" /><span className="text-sm font-medium">Require PKCE for all clients</span></label>
            <div className="relative w-20 h-20"><svg viewBox="0 0 64 64" className="w-full h-full"><circle cx={32} cy={32} r={28} fill="none" stroke="currentColor" strokeWidth={6} className="text-gray-200 dark:text-gray-800" /><circle cx={32} cy={32} r={28} fill="none" stroke={gaugeColor} strokeWidth={6} strokeDasharray={`${(config.compliance_pct / 100) * 176} 176`} strokeLinecap="round" transform="rotate(-90 32 32)" /></svg><div className="absolute inset-0 flex items-center justify-center"><span className="text-sm font-bold" style={{ color: gaugeColor }}>{config.compliance_pct.toFixed(0)}%</span></div></div>
            <span className="text-sm text-gray-500">Compliance</span>
          </div>

          {config.exempted_clients.length > 0 && (
            <div className="rounded-lg border border-yellow-300 dark:border-yellow-800 bg-yellow-50 dark:bg-yellow-900/20 p-3"><span className="text-sm font-medium text-yellow-700 dark:text-yellow-400">Exempted Clients: </span>{config.exempted_clients.map((c) => <span key={c} className="font-mono text-xs mr-2">{c}</span>)}</div>
          )}

          <div className="overflow-x-auto rounded-lg border dark:border-gray-800">
            <table className="w-full text-sm"><thead className="bg-gray-50 dark:bg-gray-900/50"><tr><th className="px-4 py-3 text-left font-medium">Client</th><th className="px-4 py-3 text-left font-medium">PKCE Required</th><th className="px-4 py-3 text-left font-medium">Challenge Method</th></tr></thead>
              <tbody className="divide-y dark:divide-gray-800">{config.per_client.map((c, i) => (<tr key={c.client_id} className="hover:bg-gray-50 dark:hover:bg-gray-900/30"><td className="px-4 py-3"><span className="font-medium">{c.client_name}</span><p className="text-xs text-gray-400 font-mono">{c.client_id}</p></td><td className="px-4 py-3"><label className="flex items-center gap-2"><input type="checkbox" checked={c.required} onChange={(e) => { const o = [...config.per_client]; o[i] = { ...c, required: e.target.checked }; setConfig({ ...config, per_client: o }); }} className="rounded" /><span className="text-xs">{c.required ? "Yes" : "No"}</span></label></td><td className="px-4 py-3"><select aria-label="Select option" value={c.challenge_method} onChange={(e) => { const o = [...config.per_client]; o[i] = { ...c, challenge_method: e.target.value as "S256" | "plain" }; setConfig({ ...config, per_client: o }); }} className="px-2 py-1 rounded border dark:border-gray-700 dark:bg-gray-900 text-xs"><option value="S256">S256</option><option value="plain">plain</option></select></td></tr>))}</tbody>
            </table>
          </div>
        </>
      )}
      {!config && !loading && <p className="text-sm text-gray-500 text-center py-8">Loading...</p>}
    </div>
  );
}
