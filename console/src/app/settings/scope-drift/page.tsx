"use client";
import { useTranslations } from "@/lib/i18n";
import { useState, useEffect, useCallback } from "react";
import { TrendingDown, AlertTriangle, Ban, Trash2 } from "lucide-react";
interface UnusedScope { scope: string; last_used_days_ago: number; severity: "low" | "medium" | "high"; }
interface DriftData { unused_scopes: UnusedScope[]; unregistered_scopes: string[]; drift_trend_30d: { date: string; value: number }[]; recommendations: string[]; }
const sevColors: Record<string, string> = { low: "bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-400", medium: "bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400", high: "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400" };
export default function ScopeDriftPage() {
  const t = useTranslations();
  const [clients] = useState([{ id: "c1", name: "Web App" }, { id: "c2", name: "Mobile App" }]);
  const [clientId, setClientId] = useState("");
  const [data, setData] = useState<DriftData | null>(null);
  const [loading, setLoading] = useState(false);
  const [revokeUnused, setRevokeUnused] = useState(false);
  const fetchData = useCallback(async () => {
    if (!clientId) return;
    setLoading(true);
    try { const res = await fetch("/api/v1/oauth/scope-drift?client_id=" + encodeURIComponent(clientId), { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } }); if (res.ok) setData(await res.json()); }
    catch { /* noop */ } finally { setLoading(false); }
  }, [clientId]);
  useEffect(() => { fetchData(); }, [fetchData]);
  const maxTrend = Math.max(...(data?.drift_trend_30d.map((d) => d.value) || [1]), 1);
  return (
    <div className="space-y-6">
      <div><h1 className="text-2xl font-bold flex items-center gap-2"><TrendingDown className="w-6 h-6 text-orange-500" />{t("scopeDrift.title")}</h1><p className="text-sm text-gray-500 mt-1">Detect unused and unregistered OAuth scopes per client.</p></div>
      <div className="flex items-center gap-3"><select aria-label="Client id" value={clientId} onChange={(e) => setClientId(e.target.value)} className="px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm"><option value="">Select Client</option>{clients.map((c) => <option key={c.id} value={c.id}>{c.name}</option>)}</select></div>
      {data && (<>
        {data.unregistered_scopes.length > 0 && <div className="rounded-lg border border-red-300 dark:border-red-800 bg-red-50 dark:bg-red-900/20 p-3"><div className="flex items-center gap-2"><AlertTriangle className="w-5 h-5 text-red-500" /><span className="font-semibold text-red-700 dark:text-red-400">Unregistered scopes detected</span></div><div className="mt-1 flex flex-wrap gap-1">{data.unregistered_scopes.map((s) => <span key={s} className="px-2 py-0.5 rounded text-xs bg-red-100 dark:bg-red-900/30 dark:text-red-400 font-mono">{s}</span>)}</div></div>}
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
          <div className="rounded-lg border dark:border-gray-800 p-4"><h3 className="text-sm font-semibold mb-3">Unused Scopes</h3><div className="space-y-1">{data.unused_scopes.map((s) => (<div key={s.scope} className="flex items-center gap-2 text-sm"><span className="font-mono text-xs flex-1">{s.scope}</span><span className="text-xs text-gray-500">{s.last_used_days_ago}d ago</span><span className={"px-2 py-0.5 rounded text-xs " + sevColors[s.severity]}>{s.severity}</span></div>))}{data.unused_scopes.length === 0 && <span className="text-xs text-gray-400">None</span>}</div><label className="flex items-center gap-2 text-sm mt-3"><input aria-label="Revoke unused" type="checkbox" checked={revokeUnused} onChange={(e) => setRevokeUnused(e.target.checked)} className="rounded" /> Revoke all unused scopes</label>{revokeUnused && <button className="mt-2 w-full px-3 py-1.5 rounded-lg bg-red-600 text-white text-xs font-medium flex items-center justify-center gap-1"><Ban className="w-3 h-3" /> Revoke Unused</button>}</div>
          <div className="rounded-lg border dark:border-gray-800 p-4"><h3 className="text-sm font-semibold mb-3">Drift Trend (30d)</h3><div className="flex items-end gap-1 h-20">{data.drift_trend_30d.map((d: any, i: number) => (<div key={i} className="flex-1 bg-orange-400 dark:bg-orange-500 rounded-t" style={{ height: (d.value / maxTrend) * 100 + "%", minHeight: "2px" }} title={d.date + ": " + d.value} />))}</div></div>
        </div>
        <div className="rounded-lg border dark:border-gray-800 p-4"><h3 className="text-sm font-semibold mb-2">Cleanup Recommendations</h3><div className="space-y-1">{data.recommendations.map((r: any, i: number) => (<div key={i} className="text-sm text-gray-500 flex items-start gap-2"><Trash2 className="w-3.5 h-3.5 text-gray-400 mt-0.5 flex-shrink-0" /> {r}</div>))}</div></div>
      </>)}
      {!data && !loading && clientId && <p className="text-sm text-gray-500 text-center py-8">Loading...</p>}
      {!clientId && <p className="text-sm text-gray-500 text-center py-8">Select a client.</p>}
    </div>
  );
}
