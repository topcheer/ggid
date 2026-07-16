"use client";
import { useTranslations } from "@/lib/i18n";
import { useState, useEffect, useCallback } from "react";
import { Activity, RefreshCw } from "lucide-react";
interface StageData { stage: string; count: number; color: string; }
interface ClientData { client_name: string; active: number; expiring: number; revoked: number; }
interface DashboardData { stages: StageData[]; avg_lifetime_hours: number; refresh_rate: number; churn_30d: { date: string; value: number }[]; issuance_rate: number; revocation_rate: number; by_client: ClientData[]; }
export default function TokenLifecycleDashboardPage() {
  const t = useTranslations();
  const [data, setData] = useState<DashboardData | null>(null);
  const [loading, setLoading] = useState(false);
  const fetchData = useCallback(async () => { setLoading(true); try { const res = await fetch("/api/v1/oauth/token-lifecycle", { headers: { "Authorization": `Bearer ${localStorage.getItem("ggid_access_token") || ""}`, "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } }); if (res.ok) setData(await res.json()); } catch { /* noop */ } finally { setLoading(false); } }, []);
  useEffect(() => { fetchData(); }, [fetchData]);
  const total = data?.stages.reduce((s, d) => s + d.count, 0) || 1;
  const maxChurn = Math.max(...(data?.churn_30d.map((d) => d.value) || [1]), 1);
  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between"><div><h1 className="text-2xl font-bold flex items-center gap-2"><Activity className="w-6 h-6 text-blue-500" /> {t("backend.tokenLifecycle.title")}</h1><p className="text-sm text-gray-500 mt-1">Monitor token lifecycle stages and churn.</p></div><button onClick={fetchData} className="px-3 py-1.5 rounded-lg border dark:border-gray-700 text-sm flex items-center gap-1"><RefreshCw className="w-3.5 h-3.5" /> Refresh</button></div>
      {data && (<>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
          <div className="rounded-lg border dark:border-gray-800 p-4"><h3 className="text-sm font-semibold mb-3">{t("backend.tokenLifecycle.tokensByStage")}</h3><div className="flex items-center gap-4"><div className="relative w-32 h-32"><svg viewBox="0 0 64 64" className="w-full h-full -rotate-90">{(() => { let offset = 0; return data.stages.map((s) => { const pct = s.count / total; const dash = pct * 176; const circle = <circle key={s.stage} cx={32} cy={32} r={28} fill="none" stroke={s.color} strokeWidth={8} strokeDasharray={dash + " 176"} strokeDashoffset={-offset * 176} />; offset += pct; return circle; }); })()}</svg></div><div className="space-y-1">{data.stages.map((s) => (<div key={s.stage} className="flex items-center gap-2 text-sm"><span className="w-3 h-3 rounded-full" style={{ background: s.color }} /><span className="capitalize">{s.stage}</span><span className="font-bold ml-auto">{s.count}</span></div>))}</div></div></div>
          <div className="rounded-lg border dark:border-gray-800 p-4"><h3 className="text-sm font-semibold mb-3">{t("backend.tokenLifecycle.metrics")}</h3><div className="grid grid-cols-2 gap-3 text-sm"><div><span className="text-xs text-gray-500">{t("backend.tokenLifecycle.avgLifetime")}</span><p className="font-bold">{data.avg_lifetime_hours}h</p></div><div><span className="text-xs text-gray-500">{t("backend.tokenLifecycle.refreshRate")}</span><p className="font-bold">{data.refresh_rate}/min</p></div><div><span className="text-xs text-gray-500">{t("backend.tokenLifecycle.issuanceRate")}</span><p className="font-bold text-green-600">{data.issuance_rate}/h</p></div><div><span className="text-xs text-gray-500">{t("backend.tokenLifecycle.revocationRate")}</span><p className="font-bold text-red-600">{data.revocation_rate}/h</p></div></div></div>
        </div>
        <div className="rounded-lg border dark:border-gray-800 p-4"><h3 className="text-sm font-semibold mb-3">Token Churn (30d)</h3><div className="flex items-end gap-1 h-24">{data.churn_30d.map((d, i) => (<div key={i} className="flex-1 bg-blue-400 dark:bg-blue-500 rounded-t" style={{ height: (d.value / maxChurn) * 100 + "%", minHeight: "2px" }} title={d.date + ": " + d.value} />))}</div></div>
        <div className="overflow-x-auto rounded-lg border dark:border-gray-800"><table className="w-full text-sm"><thead className="bg-gray-50 dark:bg-gray-900/50"><tr><th className="px-4 py-3 text-left font-medium">Client</th><th className="px-4 py-3 text-left font-medium">{t("backend.tokenLifecycle.active")}</th><th className="px-4 py-3 text-left font-medium">{t("backend.tokenLifecycle.expiring")}</th><th className="px-4 py-3 text-left font-medium">{t("backend.tokenLifecycle.revoked")}</th></tr></thead><tbody className="divide-y dark:divide-gray-800">{data.by_client.map((c, i) => (<tr key={i} className="hover:bg-gray-50 dark:hover:bg-gray-900/30"><td className="px-4 py-3 font-medium">{c.client_name}</td><td className="px-4 py-3 font-bold text-green-600">{c.active}</td><td className="px-4 py-3 text-yellow-600">{c.expiring}</td><td className="px-4 py-3 text-red-600">{c.revoked}</td></tr>))}</tbody></table></div>
      </>)}
      {!data && !loading && <p className="text-sm text-gray-500 text-center py-8">Loading...</p>}
    </div>
  );
}
