"use client";

import { useState, useEffect, useCallback } from "react";
import { Activity, Gauge, Zap, TrendingUp } from "lucide-react";

interface IntrospectionStats {
  total_requests: number;
  unique_clients: number;
  avg_latency_ms: number;
  cache_hit_rate: number;
  rate_limit_hits: number;
  top_clients: { client_id: string; client_name: string; requests: number; error_rate: number }[];
}

export default function IntrospectionStatsPage() {
  const [data, setData] = useState<IntrospectionStats | null>(null);
  const [loading, setLoading] = useState(false);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/v1/oauth/introspection-stats", { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) setData(await res.json());
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  const hitColor = data ? (data.cache_hit_rate >= 80 ? "#10b981" : data.cache_hit_rate >= 50 ? "#f59e0b" : "#ef4444") : "#3b82f6";

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><Activity className="w-6 h-6 text-purple-500" /> Introspection Stats</h1>
        <p className="text-sm text-gray-500 mt-1">Token introspection endpoint usage and performance metrics.</p>
      </div>

      {data && (
        <>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <div className="rounded-lg border p-4 dark:border-gray-800 flex items-center gap-3"><TrendingUp className="w-8 h-8 text-blue-500" /><div><span className="text-sm text-gray-500">Total Requests</span><p className="text-xl font-bold mt-1">{data.total_requests.toLocaleString()}</p></div></div>
            <div className="rounded-lg border p-4 dark:border-gray-800 flex items-center gap-3"><Activity className="w-8 h-8 text-green-500" /><div><span className="text-sm text-gray-500">Unique Clients</span><p className="text-xl font-bold mt-1">{data.unique_clients}</p></div></div>
            <div className="rounded-lg border p-4 dark:border-gray-800 flex items-center gap-3"><Zap className="w-8 h-8 text-yellow-500" /><div><span className="text-sm text-gray-500">Avg Latency</span><p className="text-xl font-bold mt-1">{data.avg_latency_ms.toFixed(1)}ms</p></div></div>
            <div className="rounded-lg border p-4 dark:border-gray-800 flex items-center gap-3"><Gauge className="w-8 h-8 text-orange-500" /><div><span className="text-sm text-gray-500">Rate Limit Hits</span><p className="text-xl font-bold text-red-600 mt-1">{data.rate_limit_hits}</p></div></div>
          </div>

          <div className="rounded-lg border dark:border-gray-800 p-4 flex items-center gap-4">
            <div className="relative w-24 h-24"><svg viewBox="0 0 64 64" className="w-full h-full"><circle cx={32} cy={32} r={28} fill="none" stroke="currentColor" strokeWidth={6} className="text-gray-200 dark:text-gray-800" /><circle cx={32} cy={32} r={28} fill="none" stroke={hitColor} strokeWidth={6} strokeDasharray={`${(data.cache_hit_rate / 100) * 176} 176`} strokeLinecap="round" transform="rotate(-90 32 32)" /></svg><div className="absolute inset-0 flex flex-col items-center justify-center"><span className="text-xl font-bold" style={{ color: hitColor }}>{data.cache_hit_rate.toFixed(0)}%</span><span className="text-[9px] text-gray-400">hit rate</span></div></div>
            <div><h3 className="font-semibold">Cache Performance</h3><p className="text-sm text-gray-500 mt-1">Introspection cache hit rate across all requests</p></div>
          </div>

          <div className="overflow-x-auto rounded-lg border dark:border-gray-800">
            <table className="w-full text-sm"><thead className="bg-gray-50 dark:bg-gray-900/50"><tr><th className="px-4 py-3 text-left font-medium">Client</th><th className="px-4 py-3 text-left font-medium">Requests</th><th className="px-4 py-3 text-left font-medium">Error Rate</th></tr></thead>
              <tbody className="divide-y dark:divide-gray-800">{data.top_clients.map((c) => (<tr key={c.client_id} className="hover:bg-gray-50 dark:hover:bg-gray-900/30"><td className="px-4 py-3"><span className="font-medium">{c.client_name}</span><p className="text-xs text-gray-400 font-mono">{c.client_id}</p></td><td className="px-4 py-3 font-bold">{c.requests.toLocaleString()}</td><td className="px-4 py-3"><span className={`font-bold ${c.error_rate > 5 ? "text-red-600" : c.error_rate > 1 ? "text-yellow-600" : "text-green-600"}`}>{c.error_rate.toFixed(2)}%</span></td></tr>))}{data.top_clients.length === 0 && <tr><td colSpan={3} className="px-4 py-8 text-center text-gray-500">No data.</td></tr>}</tbody>
            </table>
          </div>
        </>
      )}
      {!data && !loading && <p className="text-sm text-gray-500 text-center py-8">Loading...</p>}
    </div>
  );
}
