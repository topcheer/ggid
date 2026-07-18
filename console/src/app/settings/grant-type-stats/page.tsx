"use client";

import { useState, useEffect, useCallback } from "react";
import { BarChart3, PieChart as PieIcon, TrendingUp, AlertTriangle } from "lucide-react";
import { useTranslations } from "@/lib/i18n";
import { authHeader, isAuthenticated } from "@/lib/auth-helpers";

interface GrantTypeData {
  counts: { grant_type: string; count: number }[];
  trend: { date: string; [key: string]: number | string }[];
}

const typeColors: Record<string, string> = {
  authorization_code: "#3b82f6",
  client_credentials: "#8b5cf6",
  refresh_token: "#10b981",
  device_code: "#f59e0b",
};

export default function GrantTypeStatsPage() {
  const t = useTranslations();

  const [data, setData] = useState<GrantTypeData | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch("/api/v1/oauth/grant-type-stats", { headers: { ...authHeader(), "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (!res.ok) return null;
      setData(await res.json());
    } catch (e) { setError(e instanceof Error ? e.message : "Failed to load grant type stats"); }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  const total = data?.counts.reduce((s: any, d: any) => s + d.count, 0) || 1;
  const maxCount = Math.max(...(data?.counts.map((d: any) => d.count) || [1]), 1);
  const trendKeys = data ? Object.keys(data.trend[0] || {}).filter((k: any) => k !== "date") : [];
  const maxTrend = Math.max(...(data?.trend.flatMap((t) => trendKeys.map((k: any) => (t)[k] as number)) || [1]), 1);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><BarChart3 className="w-6 h-6 text-purple-500" /> {t("big1.grantTypeStats.title")}</h1>
        <p className="text-sm text-gray-500 mt-1">{t("big1.grantTypeStats.oauthGrantTypeDistributionAnd30DayTrends")}</p>
      </div>

      {error && <div className="rounded-lg border border-red-200 dark:border-red-900 bg-red-50 dark:bg-red-900/20 p-3 text-sm text-red-600 flex items-center justify-between"><span className="flex items-center gap-2"><AlertTriangle className="w-4 h-4" /> {error}</span><button aria-label="action" onClick={fetchData} className="text-xs underline hover:text-red-700">{t("big1.grantTypeStats.retry")}</button></div>}

      {loading && <div className="rounded-lg border dark:border-gray-800 p-8 text-center"><div className="inline-block w-5 h-5 border-2 border-current border-t-transparent rounded-full animate-spin text-blue-600 mb-2" /><div className="text-sm text-gray-500">{t("big1.grantTypeStats.loadingStats")}</div></div>}

      {data && (
        <>
          <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
            <div className="rounded-lg border dark:border-gray-800 p-4">
              <h3 className="text-sm font-semibold mb-3">{t("big1.grantTypeStats.barChart")}</h3>
              <div className="space-y-2">
                {data.counts.map((d: any) => (
                  <div key={d.grant_type} className="flex items-center gap-2">
                    <span className="text-xs font-mono w-32 truncate">{d.grant_type}</span>
                    <div className="flex-1 bg-gray-100 dark:bg-gray-800 rounded-full h-6 overflow-hidden"><div className="h-full rounded-full" style={{ width: `${(d.count / maxCount) * 100}%`, background: typeColors[d.grant_type] || "#ccc" }} /></div>
                    <span className="text-sm font-bold w-12 text-right">{d.count}</span>
                  </div>
                ))}
              </div>
            </div>

            <div className="rounded-lg border dark:border-gray-800 p-4">
              <h3 className="text-sm font-semibold mb-3 flex items-center gap-2"><PieIcon className="w-4 h-4" />{t("big1.grantTypeStats.percentage")}</h3>
              <div className="flex items-center gap-4">
                <div className="relative w-24 h-24">
                  <svg viewBox="0 0 64 64" className="w-full h-full -rotate-90">
                    {(() => { let off = 0; return data.counts.map((d: any) => { const pct = d.count / total; const dash = pct * 176; const c = <circle key={d.grant_type} cx={32} cy={32} r={28} fill="none" stroke={typeColors[d.grant_type] || "#ccc"} strokeWidth={8} strokeDasharray={`${dash} 176`} strokeDashoffset={-off * 176} />; off += pct; return c; }); })()}
                  </svg>
                </div>
                <div className="space-y-1">{data.counts.map((d: any) => (<div key={d.grant_type} className="flex items-center gap-2 text-xs">
                  <span className="w-3 h-3 rounded" style={{ background: typeColors[d.grant_type] || "#ccc" }} />
                  <span className="flex-1 truncate">{d.grant_type}</span>
                  <span className="font-bold">{((d.count / total) * 100).toFixed(1)}%</span>
                </div>))}</div>
              </div>
            </div>

            <div className="rounded-lg border dark:border-gray-800 p-4">
              <h3 className="text-sm font-semibold mb-3 flex items-center gap-2"><TrendingUp className="w-4 h-4" />{t("big1.grantTypeStats.30DayTrend")}</h3>
              <svg viewBox="0 0 200 60" className="w-full h-16">
                {trendKeys.map((key: any) => { const pts = data.trend.map((t: any, i: number) => `${(i / (data.trend.length - 1 || 1)) * 200},${55 - ((t)[key] as number / maxTrend) * 50}`).join(" "); return <polyline key={key} fill="none" stroke={typeColors[key] || "#ccc"} strokeWidth={1.5} points={pts} />; })}
              </svg>
              <div className="flex flex-wrap gap-2 mt-2">{trendKeys.map((k: any) => <span key={k} className="flex items-center gap-1 text-xs"><span className="w-2 h-2 rounded" style={{ background: typeColors[k] || "#ccc" }} />{k}</span>)}</div>
            </div>
          </div>

          <div className="overflow-x-auto rounded-lg border dark:border-gray-800">
            <table className="w-full text-sm">
              <thead className="bg-gray-50 dark:bg-gray-900/50"><tr><th className="px-4 py-3 text-left font-medium">{t("big1.grantTypeStats.grantType")}</th><th className="px-4 py-3 text-left font-medium">{t("big1.grantTypeStats.count")}</th><th className="px-4 py-3 text-left font-medium">{t("big1.grantTypeStats.share")}</th></tr></thead>
              <tbody className="divide-y dark:divide-gray-800">{data.counts.map((d: any) => (<tr key={d.grant_type} className="hover:bg-gray-50 dark:hover:bg-gray-900/30">
                <td className="px-4 py-3 font-mono text-xs">{d.grant_type}</td><td className="px-4 py-3 font-bold">{d.count}</td><td className="px-4 py-3 text-gray-500">{((d.count / total) * 100).toFixed(1)}%</td>
              </tr>))}</tbody>
            </table>
          </div>
        </>
      )}
    </div>
  );
}
