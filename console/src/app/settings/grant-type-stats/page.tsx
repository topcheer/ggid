"use client";

import { useState, useEffect, useCallback } from "react";
import { BarChart3, PieChart as PieIcon, TrendingUp } from "lucide-react";

interface GrantTypeData {
  counts: { grant_type: string; count: number }[];
  trend: { date: string; authorization_code: number; client_credentials: number; refresh_token: number; device_code: number }[];
}

const typeColors: Record<string, string> = {
  authorization_code: "#3b82f6",
  client_credentials: "#8b5cf6",
  refresh_token: "#10b981",
  device_code: "#f59e0b",
};

export default function GrantTypeStatsPage() {
  const [data, setData] = useState<GrantTypeData | null>(null);
  const [loading, setLoading] = useState(false);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/v1/oauth/grant-type-stats", { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) setData(await res.json());
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  const total = data?.counts.reduce((s, d) => s + d.count, 0) || 1;
  const maxCount = Math.max(...(data?.counts.map((d) => d.count) || [1]), 1);
  const trendKeys = data ? Object.keys(data.trend[0] || {}).filter((k) => k !== "date") : [];
  const maxTrend = Math.max(...(data?.trend.flatMap((t) => trendKeys.map((k) => (t as any)[k])) || [1]), 1);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><BarChart3 className="w-6 h-6 text-purple-500" /> Grant Type Statistics</h1>
        <p className="text-sm text-gray-500 mt-1">OAuth grant type distribution and 30-day trends.</p>
      </div>

      {data && (
        <>
          <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
            <div className="rounded-lg border dark:border-gray-800 p-4">
              <h3 className="text-sm font-semibold mb-3">Bar Chart</h3>
              <div className="space-y-2">
                {data.counts.map((d) => (
                  <div key={d.grant_type} className="flex items-center gap-2">
                    <span className="text-xs font-mono w-32 truncate">{d.grant_type}</span>
                    <div className="flex-1 bg-gray-100 dark:bg-gray-800 rounded-full h-6 overflow-hidden"><div className="h-full rounded-full" style={{ width: `${(d.count / maxCount) * 100}%`, background: typeColors[d.grant_type] || "#ccc" }} /></div>
                    <span className="text-sm font-bold w-12 text-right">{d.count}</span>
                  </div>
                ))}
              </div>
            </div>

            <div className="rounded-lg border dark:border-gray-800 p-4">
              <h3 className="text-sm font-semibold mb-3 flex items-center gap-2"><PieIcon className="w-4 h-4" /> Percentage</h3>
              <div className="flex items-center gap-4">
                <div className="relative w-24 h-24"><svg viewBox="0 0 64 64" className="w-full h-full -rotate-90">{(() => { let off = 0; return data.counts.map((d) => { const pct = d.count / total; const dash = pct * 176; const c = <circle key={d.grant_type} cx={32} cy={32} r={28} fill="none" stroke={typeColors[d.grant_type] || "#ccc"} strokeWidth={8} strokeDasharray={`${dash} 176`} strokeDashoffset={-off * 176} />; off += pct; return c; }); })()}</svg></div>
                <div className="space-y-1">{data.counts.map((d) => (<div key={d.grant_type} className="flex items-center gap-2 text-xs"><span className="w-3 h-3 rounded" style={{ background: typeColors[d.grant_type] || "#ccc" }} /><span className="flex-1 truncate">{d.grant_type}</span><span className="font-bold">{((d.count / total) * 100).toFixed(1)}%</span></div>))}</div>
              </div>
            </div>

            <div className="rounded-lg border dark:border-gray-800 p-4">
              <h3 className="text-sm font-semibold mb-3 flex items-center gap-2"><TrendingUp className="w-4 h-4" /> 30-Day Trend</h3>
              <svg viewBox="0 0 200 60" className="w-full h-16">
                {trendKeys.map((key) => { const pts = data.trend.map((t, i) => `${(i / (data.trend.length - 1 || 1)) * 200},${55 - ((t as any)[key] / maxTrend) * 50}`).join(" "); return <polyline key={key} fill="none" stroke={typeColors[key] || "#ccc"} strokeWidth={1.5} points={pts} />; })}
              </svg>
              <div className="flex flex-wrap gap-2 mt-2">{trendKeys.map((k) => <span key={k} className="flex items-center gap-1 text-xs"><span className="w-2 h-2 rounded" style={{ background: typeColors[k] || "#ccc" }} />{k}</span>)}</div>
            </div>
          </div>

          <div className="overflow-x-auto rounded-lg border dark:border-gray-800">
            <table className="w-full text-sm">
              <thead className="bg-gray-50 dark:bg-gray-900/50"><tr><th className="px-4 py-3 text-left font-medium">Grant Type</th><th className="px-4 py-3 text-left font-medium">Count</th><th className="px-4 py-3 text-left font-medium">Share</th></tr></thead>
              <tbody className="divide-y dark:divide-gray-800">{data.counts.map((d) => (<tr key={d.grant_type} className="hover:bg-gray-50 dark:hover:bg-gray-900/30"><td className="px-4 py-3 font-mono text-xs">{d.grant_type}</td><td className="px-4 py-3 font-bold">{d.count}</td><td className="px-4 py-3 text-gray-500">{((d.count / total) * 100).toFixed(1)}%</td></tr>))}</tbody>
            </table>
          </div>
        </>
      )}
      {!data && !loading && <p className="text-sm text-gray-500 text-center py-8">Loading...</p>}
    </div>
  );
}
