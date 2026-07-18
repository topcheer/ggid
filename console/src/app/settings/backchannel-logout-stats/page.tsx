"use client";

import { useState, useEffect, useCallback } from "react";
import { LogOut, CheckCircle, XCircle } from "lucide-react";
import { useTranslations } from "@/lib/i18n";
import { authHeader, isAuthenticated } from "@/lib/auth-helpers";

interface Stats {
  total_requests: number;
  successful_pct: number;
  failed_count: number;
  top_failure_reasons: { reason: string; count: number }[];
  avg_latency_ms: number;
  by_idp_provider: { provider: string; requests: number; success_pct: number }[];
}

export default function BackchannelLogoutStatsPage() {
  const t = useTranslations();

  const [data, setData] = useState<Stats | null>(null);
  const [loading, setLoading] = useState(false);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/v1/auth/backchannel-logout-stats", { headers: { ...authHeader(), "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) setData(await res.json());
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  const gaugeColor = data ? (data.successful_pct >= 95 ? "#10b981" : data.successful_pct >= 80 ? "#f59e0b" : "#ef4444") : "#3b82f6";

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><LogOut className="w-6 h-6 text-purple-500" /> {t("backchannelLogoutStats.title")}</h1>
        <p className="text-sm text-gray-500 mt-1">OIDC backchannel logout endpoint statistics and failure analysis.</p>
      </div>

      {data && (
        <>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">Total Requests</span><p className="text-xl font-bold mt-1">{data.total_requests.toLocaleString()}</p></div>
            <div className="rounded-lg border p-4 dark:border-gray-800 flex items-center gap-4">
              <div className="relative w-16 h-16"><svg viewBox="0 0 64 64" className="w-full h-full"><circle cx={32} cy={32} r={28} fill="none" stroke="currentColor" strokeWidth={6} className="text-gray-200 dark:text-gray-800" /><circle cx={32} cy={32} r={28} fill="none" stroke={gaugeColor} strokeWidth={6} strokeDasharray={`${(data.successful_pct / 100) * 176} 176`} strokeLinecap="round" transform="rotate(-90 32 32)" /></svg><div className="absolute inset-0 flex items-center justify-center"><span className="text-sm font-bold" style={{ color: gaugeColor }}>{data.successful_pct.toFixed(0)}%</span></div></div>
              <div><span className="text-sm text-gray-500">Success Rate</span></div>
            </div>
            <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">Failed</span><p className="text-xl font-bold text-red-600 mt-1">{data.failed_count}</p></div>
            <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">Avg Latency</span><p className="text-xl font-bold mt-1">{data.avg_latency_ms}ms</p></div>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            <div className="rounded-lg border dark:border-gray-800 p-4"><h3 className="text-sm font-semibold mb-3">Top Failure Reasons</h3><div className="space-y-2">{data.top_failure_reasons.map((f: any, i: number) => (<div key={i} className="flex items-center gap-2"><XCircle className="w-4 h-4 text-red-400" /><span className="text-xs flex-1">{f.reason}</span><span className="font-bold text-red-600 text-sm">{f.count}</span></div>))}{data.top_failure_reasons.length === 0 && <p className="text-xs text-gray-400">No failures recorded.</p>}</div></div>
            <div className="rounded-lg border dark:border-gray-800 p-4"><h3 className="text-sm font-semibold mb-3">By IdP Provider</h3><div className="space-y-2">{data.by_idp_provider.map((p: any) => (<div key={p.provider} className="flex items-center gap-2"><span className="text-xs w-24">{p.provider}</span><div className="flex-1 bg-gray-100 dark:bg-gray-800 rounded-full h-4 overflow-hidden"><div className={"h-full rounded-full " + (p.success_pct >= 95 ? "bg-green-500" : p.success_pct >= 80 ? "bg-yellow-500" : "bg-red-500")} style={{ width: p.success_pct + "%" }} /></div><span className="text-xs font-bold w-10 text-right">{p.success_pct.toFixed(0)}%</span></div>))}</div></div>
          </div>
        </>
      )}
      {!data && !loading && <p className="text-sm text-gray-500 text-center py-8">Loading...</p>}
    </div>
  );
}
