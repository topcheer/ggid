import { useState, useCallback } from "react";

export interface IntrospectionStats {
  total_requests: number;
  unique_clients: number;
  avg_latency_ms: number;
  cache_hit_rate: number;
  rate_limit_hits: number;
  top_clients: { client_id: string; client_name: string; requests: number; error_rate: number }[];
}

export function useIntrospectionStats(baseUrl: string = "") {
  const [data, setData] = useState<IntrospectionStats | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchStats = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/oauth/introspection-stats`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setData(await res.json());
    } catch (e: any) { setError(e.message); }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { data, loading, error, fetchStats };
}
