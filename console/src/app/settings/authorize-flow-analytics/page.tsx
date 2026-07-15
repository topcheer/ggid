"use client";

import { useState, useEffect, useCallback } from "react";
import { Activity, TrendingDown, AlertTriangle } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface Analytics {
  total_attempts: number;
  consent_rate: number;
  abandonment_at_step: { step: string; count: number; pct: number }[];
  avg_duration_ms: number;
  top_clients: { client_id: string; client_name: string; attempts: number; success_pct: number }[];
  pkce_adoption_pct: number;
  redirect_uri_errors: number;
}

export default function AuthorizeFlowAnalyticsPage() {
  const t = useTranslations();

  const [data, setData] = useState<Analytics | null>(null);
  const [loading, setLoading] = useState(false);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/v1/oauth/authorize-flow-analytics", { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) setData(await res.json());
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  const gaugeColor = data ? (data.consent_rate >= 80 ? "#10b981" : data.consent_rate >= 50 ? "#f59e0b" : "#ef4444") : "#3b82f6";
  const maxAbandon = Math.max(...(data?.abandonment_at_step.map((a) => a.count) || [1]), 1);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><Activity className="w-6 h-6 text-blue-500" /> {t("authorizeFlowAnalytics.title")}</h1>
        <p className="text-sm text-gray-500 mt-1">OAuth authorize endpoint analytics with consent rates and abandonment funnel.</p>
      </div>

      {data && (
        <>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">Total Attempts</span><p className="text-xl font-bold mt-1">{data.total_attempts.toLocaleString()}</p></div>
            <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">Avg Duration</span><p className="text-xl font-bold mt-1">{data.avg_duration_ms}ms</p></div>
            <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">PKCE Adoption</span><div className="flex items-center gap-2 mt-1"><div className="w-20 bg-gray-100 dark:bg-gray-800 rounded-full h-3 overflow-hidden"><div className="h-full bg-green-500 rounded-full" style={{ width: `${data.pkce_adoption_pct}%` }} /></div><span className="font-bold text-green-600 text-sm">{data.pkce_adoption_pct.toFixed(0)}%</span></div></div>
            <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">Redirect URI Errors</span><p className="text-xl font-bold text-red-600 mt-1">{data.redirect_uri_errors}</p></div>
          </div>

          <div className="rounded-lg border dark:border-gray-800 p-4 flex items-center gap-4">
            <div className="relative w-24 h-24"><svg viewBox="0 0 64 64" className="w-full h-full"><circle cx={32} cy={32} r={28} fill="none" stroke="currentColor" strokeWidth={6} className="text-gray-200 dark:text-gray-800" /><circle cx={32} cy={32} r={28} fill="none" stroke={gaugeColor} strokeWidth={6} strokeDasharray={`${(data.consent_rate / 100) * 176} 176`} strokeLinecap="round" transform="rotate(-90 32 32)" /></svg><div className="absolute inset-0 flex flex-col items-center justify-center"><span className="text-lg font-bold" style={{ color: gaugeColor }}>{data.consent_rate.toFixed(0)}%</span><span className="text-[9px] text-gray-400">consent</span></div></div>
            <div><h3 className="font-semibold">Consent Rate</h3><p className="text-sm text-gray-500 mt-1">Users who granted consent after seeing the prompt</p></div>
          </div>

          <div className="rounded-lg border dark:border-gray-800 p-4">
            <h3 className="text-sm font-semibold flex items-center gap-2 mb-3"><TrendingDown className="w-4 h-4 text-red-500" /> Abandonment Funnel</h3>
            <div className="space-y-2">{data.abandonment_at_step.map((s) => (
              <div key={s.step} className="flex items-center gap-2"><span className="text-xs text-gray-500 w-32">{s.step}</span><div className="flex-1 bg-gray-100 dark:bg-gray-800 rounded-full h-5 overflow-hidden"><div className="h-full bg-red-500 rounded-full" style={{ width: `${(s.count / maxAbandon) * 100}%` }} /></div><span className="text-xs font-bold w-20 text-right">{s.count} ({s.pct.toFixed(1)}%)</span></div>
            ))}</div>
          </div>

          <div className="overflow-x-auto rounded-lg border dark:border-gray-800">
            <table className="w-full text-sm"><thead className="bg-gray-50 dark:bg-gray-900/50"><tr><th className="px-4 py-3 text-left font-medium">Client</th><th className="px-4 py-3 text-left font-medium">Attempts</th><th className="px-4 py-3 text-left font-medium">Success Rate</th></tr></thead>
              <tbody className="divide-y dark:divide-gray-800">{data.top_clients.map((c) => (<tr key={c.client_id} className="hover:bg-gray-50 dark:hover:bg-gray-900/30"><td className="px-4 py-3"><span className="font-medium">{c.client_name}</span><p className="text-xs text-gray-400 font-mono">{c.client_id}</p></td><td className="px-4 py-3 font-bold">{c.attempts.toLocaleString()}</td><td className="px-4 py-3"><span className={`font-bold ${c.success_pct >= 80 ? "text-green-600" : c.success_pct >= 50 ? "text-yellow-600" : "text-red-600"}`}>{c.success_pct.toFixed(1)}%</span></td></tr>))}</tbody>
            </table>
          </div>
        </>
      )}
      {!data && !loading && <p className="text-sm text-gray-500 text-center py-8">Loading...</p>}
    </div>
  );
}
