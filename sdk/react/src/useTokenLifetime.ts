import { useState, useCallback } from "react";

export interface LifetimeData {
  avg_lifetime_minutes: number;
  median_lifetime_minutes: number;
  short_lived_count: number;
  long_lived_count: number;
  distribution: { range: string; count: number }[];
  per_client: { client_id: string; client_name: string; avg_minutes: number; token_count: number; short_pct: number }[];
}

export function useTokenLifetime(baseUrl: string = "") {
  const [data, setData] = useState<LifetimeData | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchLifetime = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/oauth/token-lifetime`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const json: LifetimeData = await res.json();
      setData(json);
    } catch (e: any) {
      setError(e.message);
      setData(null);
    } finally {
      setLoading(false);
    }
  }, [baseUrl]);

  return { data, loading, error, fetchLifetime };
}
