import { useState, useCallback } from "react";
export interface DbHealth { pool_size: number; pool_active: number; pool_idle: number; pool_max: number; query_rate_per_sec: number; avg_latency_ms: number; slow_queries: { query: string; duration_ms: number; timestamp: string }[]; table_sizes: { table: string; size_mb: number }[]; index_efficiency_pct: number; replication_lag_seconds: number; }
export function useDatabaseHealth(baseUrl: string = "") {
  const [data, setData] = useState<DbHealth | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const fetchData = useCallback(async () => { setLoading(true); setError(null); try { const res = await fetch(baseUrl + "/api/v1/admin/database-health"); if (!res.ok) throw new Error("HTTP " + res.status); setData(await res.json()); } catch (e: any) { setError(e.message); } finally { setLoading(false); } }, [baseUrl]);
  return { data, loading, error, fetchData };
}
