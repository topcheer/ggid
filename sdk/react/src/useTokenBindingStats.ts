import { useState, useCallback } from "react";

export interface BindingStats {
  bound: number;
  unbound: number;
  total: number;
  compliance_pct: number;
  binding_methods: { method: string; count: number }[];
  by_client: { client_id: string; client_name: string; bound: number; unbound: number; method: string }[];
}

export function useTokenBindingStats(baseUrl: string = "") {
  const [data, setData] = useState<BindingStats | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchStats = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/oauth/token-binding-stats`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setData(await res.json());
    } catch (e: any) { setError(e.message); }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { data, loading, error, fetchStats };
}
