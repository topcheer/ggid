import { useState, useCallback } from "react";
export interface CacheInstance { id: string; name: string; type: string; status: string; hit_rate_pct: number; memory_used_mb: number; memory_max_mb: number; keys: number; evictions_per_min: number; latency_ms: number; top_keys: { key: string; hits: number; ttl: number }[]; }
export function useCacheHealth(baseUrl: string = "") {
  const [instances, setInstances] = useState<CacheInstance[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const fetchData = useCallback(async () => { setLoading(true); setError(null); try { const res = await fetch(baseUrl + "/api/v1/admin/cache-health"); if (!res.ok) throw new Error("HTTP " + res.status); const d = await res.json(); setInstances(d.instances || d || []); } catch (e: any) { setError(e.message); } finally { setLoading(false); } }, [baseUrl]);
  return { instances, loading, error, fetchData };
}
