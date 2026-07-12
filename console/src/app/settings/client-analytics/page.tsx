"use client";

import { useState, useEffect, useCallback } from "react";
import { BarChart3, Gauge, Users, Zap } from "lucide-react";

interface ClientData {
  token_usage_30d: { day: string; count: number }[];
  active_tokens: number;
  total_tokens_issued: number;
  error_rate: number;
  avg_latency_ms: number;
  top_users: { user_id: string; username: string; requests: number; last_active: string }[];
  scopes_requested: { scope: string; count: number }[];
}

interface Client { client_id: string; client_name: string; }

export default function ClientAnalyticsPage() {
  const [clients] = useState<Client[]>([{ client_id: "c1", client_name: "Web App" }, { client_id: "c2", client_name: "Mobile App" }, { client_id: "c3", client_name: "API Gateway" }]);
  const [selectedId, setSelectedId] = useState("");
  const [data, setData] = useState<ClientData | null>(null);
  const [loading, setLoading] = useState(false);

  const fetchData = useCallback(async () => {
    if (!selectedId) return;
    setLoading(true);
    try { const res = await fetch("/api/v1/oauth/client-analytics?client_id=" + encodeURIComponent(selectedId), { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } }); if (res.ok) setData(await res.json()); }
    catch { /* noop */ }
    finally { setLoading(false); }
  }, [selectedId]);

  useEffect(() => { fetchData(); }, [fetchData]);

  const maxUsage = Math.max(...(data?.token_usage_30d.map((t) => t.count) || [1]), 1);
  const maxScope = Math.max(...(data?.scopes_requested.map((s) => s.count) || [1]), 1);

  return (
    <div className="space-y-6">
      <div><h1 className="text-2xl font-bold flex items-center gap-2"><BarChart3 className="w-6 h-6 text-blue-500" /> Client Analytics</h1><p className="text-sm text-gray-500 mt-1">OAuth client usage patterns and performance metrics.</p></div>

      <select value={selectedId} onChange={(e) => setSelectedId(e.target.value)} className="px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm"><option value="">Select Client</option>{clients.map((c) => <option key={c.client_id} value={c.client_id}>{c.client_name}</option>)}</select>

      {data && (
        <>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <div className="rounded-lg border p-4 dark:border-gray-800 flex items-center gap-3"><Gauge className="w-8 h-8 text-green-500" /><div><span className="text-sm text-gray-500">Active Tokens</span><p className="text-xl font-bold">{data.active_tokens}</p></div></div>
            <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">Total Issued</span><p className="text-xl font-bold mt-1">{data.total_tokens_issued.toLocaleString()}</p></div>
            <div className="rounded-lg border p-4 dark:border-gray-800 flex items-center gap-3"><Zap className="w-8 h-8 text-orange-500" /><div><span className="text-sm text-gray-500">Error Rate</span><p className={"text-xl font-bold " + (data.error_rate > 5 ? "text-red-600" : "text-green-600")}>{data.error_rate.toFixed(2)}%</p></div></div>
            <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">Avg Latency</span><p className="text-xl font-bold mt-1">{data.avg_latency_ms}ms</p></div>
          </div>

          <div className="rounded-lg border dark:border-gray-800 p-4"><h3 className="text-sm font-semibold mb-3">Token Usage (30d)</h3><div className="flex items-end gap-0.5 h-24">{data.token_usage_30d.map((t, i) => <div key={i} className="flex-1 bg-blue-400 dark:bg-blue-500 rounded-t" style={{ height: (t.count / maxUsage) * 100 + "%", minHeight: "2px" }} title={t.day + ": " + t.count} />)}</div></div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            <div className="rounded-lg border dark:border-gray-800 p-4"><h3 className="text-sm font-semibold flex items-center gap-2 mb-3"><Users className="w-4 h-4 text-gray-400" /> Top Users</h3><div className="space-y-2">{data.top_users.map((u) => <div key={u.user_id} className="flex items-center gap-2"><span className="text-sm font-medium flex-1">{u.username}</span><span className="text-xs text-gray-500">{u.last_active}</span><span className="px-2 py-0.5 rounded text-xs bg-blue-100 dark:bg-blue-900/30 dark:text-blue-400 font-bold">{u.requests}</span></div>)}</div></div>
            <div className="rounded-lg border dark:border-gray-800 p-4"><h3 className="text-sm font-semibold mb-3">Scopes Heatmap</h3><div className="space-y-1">{data.scopes_requested.map((s) => <div key={s.scope} className="flex items-center gap-2"><span className="text-xs font-mono w-32 truncate">{s.scope}</span><div className="flex-1 bg-gray-100 dark:bg-gray-800 rounded-full h-4 overflow-hidden"><div className={"h-full rounded-full " + (s.count / maxScope > 0.7 ? "bg-red-500" : s.count / maxScope > 0.4 ? "bg-yellow-500" : "bg-green-500")} style={{ width: (s.count / maxScope) * 100 + "%" }} /></div><span className="text-xs font-bold w-10 text-right">{s.count}</span></div>)}</div></div>
          </div>
        </>
      )}
      {!data && !loading && selectedId && <p className="text-sm text-gray-500 text-center py-8">Loading...</p>}
      {!selectedId && <p className="text-sm text-gray-500 text-center py-8">Select a client to view analytics.</p>}
    </div>
  );
}
