"use client";
import { useTranslations } from "@/lib/i18n";
import { useState, useEffect, useCallback } from "react";
import { FilePlus, Save } from "lucide-react";
import { authHeader, isAuthenticated } from "@/lib/auth-helpers";
interface ParOverride { client_id: string; client_name: string; required: boolean; }
interface ParStats { total: number; success_rate: number; avg_latency_ms: number; }
interface Config { require_par: boolean; par_lifetime_seconds: number; max_request_size_kb: number; per_client: ParOverride[]; exempted_clients: string[]; stats: ParStats; }
export default function ParConfigPage() {
  const t = useTranslations();
  const [config, setConfig] = useState<Config>({ require_par: false, par_lifetime_seconds: 60, max_request_size_kb: 32, per_client: [{ client_id: "c1", client_name: "Web App", required: false }], exempted_clients: [], stats: { total: 1247, success_rate: 98.5, avg_latency_ms: 45 } });
  const [saving, setSaving] = useState(false);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const loadData = useCallback(async () => {
    setLoading(true); setError(null);
    try { const res = await fetch("/api/v1/oauth/par-config", { headers: { ...authHeader(), "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } }); if (res.ok) { const d = await res.json(); if (d) setConfig(prev => ({ ...prev, ...d })); } }
    catch (err) { setError(err instanceof Error ? err.message : "An error occurred"); } finally { setLoading(false); }
  }, []);
  useEffect(() => { loadData(); }, [loadData]);
  const save = useCallback(async () => { setSaving(true); try { await fetch("/api/v1/oauth/par-config", { method: "PUT", headers: { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" }, body: JSON.stringify(config) }); } catch { /* noop */ } finally { setSaving(false); } }, [config]);
  const gaugeColor = config.stats.success_rate >= 95 ? "#10b981" : config.stats.success_rate >= 80 ? "#f59e0b" : "#ef4444";
  if (loading) return (<div className="p-8 flex items-center justify-center"><div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600" /></div>);
  if (error) return (<div className="p-8"><div className="rounded-lg border border-red-300 bg-red-50 dark:bg-red-950 dark:border-red-800 p-4"><p className="text-red-700 dark:text-red-400 text-sm font-medium">Error: {error}</p><button onClick={loadData} className="mt-2 px-4 py-1.5 rounded-lg bg-red-600 text-white text-sm hover:bg-red-700">Retry</button></div></div>);
  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between"><div><h1 className="text-2xl font-bold flex items-center gap-2"><FilePlus className="w-6 h-6 text-blue-500" />{t("parConfig.title")}</h1><p className="text-sm text-gray-500 mt-1">Pushed Authorization Requests (RFC 9126) configuration.</p></div><button onClick={save} disabled={saving} className="px-4 py-2 rounded-lg bg-blue-600 text-white text-sm font-medium flex items-center gap-2"><Save className="w-4 h-4" /> Save</button></div>
      <div className="rounded-lg border dark:border-gray-800 p-4 space-y-4 max-w-lg"><label className="flex items-center gap-3 cursor-pointer"><input aria-label="Config" type="checkbox" checked={config.require_par} onChange={(e) => setConfig({ ...config, require_par: e.target.checked })} className="rounded" /><span className="text-sm font-medium">Require PAR for all clients</span></label><div><label className="text-sm font-medium">PAR Lifetime (seconds)</label><div className="flex items-center gap-3 mt-1"><input type="range" min={10} max={300} value={config.par_lifetime_seconds} onChange={(e) => setConfig({ ...config, par_lifetime_seconds: parseInt(e.target.value) })} className="flex-1" /><span className="text-sm font-bold w-12">{config.par_lifetime_seconds}s</span></div></div><div><label className="text-sm font-medium">Max Request Size (KB)</label><input type="number" min={1} max={256} value={config.max_request_size_kb} onChange={(e) => setConfig({ ...config, max_request_size_kb: parseInt(e.target.value) })} className="w-24 mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm" /></div></div>
      <div className="grid grid-cols-3 gap-4"><div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">Total PAR</span><p className="text-xl font-bold mt-1">{config.stats.total.toLocaleString()}</p></div><div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">Success Rate</span><p className="text-xl font-bold mt-1" style={{ color: gaugeColor }}>{config.stats.success_rate}%</p></div><div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">Avg Latency</span><p className="text-xl font-bold mt-1">{config.stats.avg_latency_ms}ms</p></div></div>
      {config.exempted_clients.length > 0 && <div className="rounded-lg border border-yellow-300 dark:border-yellow-800 bg-yellow-50 dark:bg-yellow-900/20 p-3"><span className="text-sm font-medium text-yellow-700 dark:text-yellow-400">Exempted: </span>{config.exempted_clients.map((c: any) => <span key={c} className="font-mono text-xs mr-2">{c}</span>)}</div>}
      <div className="overflow-x-auto rounded-lg border dark:border-gray-800"><table className="w-full text-sm"><thead className="bg-gray-50 dark:bg-gray-900/50"><tr><th className="px-4 py-3 text-left font-medium">Client</th><th className="px-4 py-3 text-left font-medium">PAR Required</th></tr></thead><tbody className="divide-y dark:divide-gray-800">{config.per_client.map((c: any, i: number) => (<tr key={c.client_id} className="hover:bg-gray-50 dark:hover:bg-gray-900/30"><td className="px-4 py-3"><span className="font-medium">{c.client_name}</span></td><td className="px-4 py-3"><label className="flex items-center gap-2"><input aria-label="C" type="checkbox" checked={c.required} onChange={(e) => { const o = [...config.per_client]; o[i] = { ...c, required: e.target.checked }; setConfig({ ...config, per_client: o }); }} className="rounded" /><span className="text-xs">{c.required ? "Yes" : "No"}</span></label></td></tr>))}</tbody></table></div>
    </div>
  );
}
