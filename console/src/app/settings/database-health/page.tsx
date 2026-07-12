"use client";
import { useState, useEffect, useCallback } from "react";
import { Database, Clock, AlertTriangle } from "lucide-react";

interface DbHealth { pool_size: number; pool_active: number; pool_idle: number; pool_max: number; query_rate_per_sec: number; avg_latency_ms: number; slow_queries: { query: string; duration_ms: number; timestamp: string }[]; table_sizes: { table: string; size_mb: number }[]; index_efficiency_pct: number; replication_lag_seconds: number; }

export default function DatabaseHealthPage() {
  const [data, setData] = useState<DbHealth | null>(null);
  const [loading, setLoading] = useState(false);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/v1/admin/database-health", { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) setData(await res.json());
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  if (!data) return <p className="text-sm text-gray-500 text-center py-8">Loading...</p>;

  const poolPct = (data.pool_active / data.pool_max) * 100;
  const maxSize = Math.max(...data.table_sizes.map((t) => t.size_mb), 1);

  return (
    <div className="space-y-6">
      <div><h1 className="text-2xl font-bold flex items-center gap-2"><Database className="w-6 h-6 text-blue-500" /> Database Health</h1><p className="text-sm text-gray-500 mt-1">Monitor connection pool, query performance, and storage.</p></div>

      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">Query Rate</span><p className="text-xl font-bold mt-1">{data.query_rate_per_sec}/s</p></div>
        <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">Avg Latency</span><p className={"text-xl font-bold mt-1 " + (data.avg_latency_ms > 100 ? "text-red-600" : "text-green-600")}>{data.avg_latency_ms}ms</p></div>
        <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">Index Efficiency</span><p className="text-xl font-bold mt-1">{data.index_efficiency_pct}%</p></div>
        <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">Repl Lag</span><p className={"text-xl font-bold mt-1 " + (data.replication_lag_seconds > 5 ? "text-red-600" : "text-green-600")}>{data.replication_lag_seconds}s</p></div>
      </div>

      <div className="rounded-lg border dark:border-gray-800 p-4"><div className="flex items-center justify-between mb-3"><h3 className="text-sm font-semibold">Connection Pool</h3><span className="text-sm text-gray-500">{data.pool_active}/{data.pool_max} active</span></div><div className="w-full bg-gray-100 dark:bg-gray-800 rounded-full h-6 overflow-hidden"><div className={"h-full rounded-full " + (poolPct > 80 ? "bg-red-500" : poolPct > 60 ? "bg-yellow-500" : "bg-green-500")} style={{ width: poolPct + "%" }} /></div><div className="flex justify-between mt-2 text-xs text-gray-500"><span>Active: {data.pool_active}</span><span>Idle: {data.pool_idle}</span><span>Max: {data.pool_max}</span></div>{poolPct > 80 && <div className="mt-2 flex items-center gap-2 text-xs text-red-600"><AlertTriangle className="w-3.5 h-3.5" /> Pool utilization above 80%</div>}</div>

      <div className="rounded-lg border dark:border-gray-800 p-4"><h3 className="text-sm font-semibold mb-3 flex items-center gap-2"><Clock className="w-4 h-4 text-gray-400" /> Slow Queries</h3><div className="space-y-1">{data.slow_queries.map((q, i) => (<div key={i} className="flex items-center gap-2 text-sm py-1"><span className={"px-2 py-0.5 rounded text-xs font-bold " + (q.duration_ms > 1000 ? "bg-red-100 dark:bg-red-900/30 dark:text-red-400" : "bg-yellow-100 dark:bg-yellow-900/30 dark:text-yellow-400")}>{q.duration_ms}ms</span><span className="font-mono text-xs text-gray-500 truncate flex-1">{q.query.substring(0, 80)}</span><span className="text-xs text-gray-400">{q.timestamp}</span></div>))}{data.slow_queries.length === 0 && <p className="text-xs text-gray-500">No slow queries.</p>}</div></div>

      <div className="rounded-lg border dark:border-gray-800 p-4"><h3 className="text-sm font-semibold mb-3">Table Sizes</h3><div className="space-y-2">{data.table_sizes.map((t) => (<div key={t.table} className="flex items-center gap-2"><span className="text-xs text-gray-500 w-32 truncate font-mono">{t.table}</span><div className="flex-1 bg-gray-100 dark:bg-gray-800 rounded-full h-4"><div className="h-full rounded-full bg-blue-500" style={{ width: (t.size_mb / maxSize) * 100 + "%" }} /></div><span className="text-xs font-bold w-16 text-right">{t.size_mb.toFixed(1)} MB</span></div>))}</div></div>
    </div>
  );
}
