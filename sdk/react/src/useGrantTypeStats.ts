import { useState, useCallback } from "react";

export interface GrantTypeData {
  counts: { grant_type: string; count: number }[];
  trend: { date: string; authorization_code: number; client_credentials: number; refresh_token: number; device_code: number }[];
}

export function useGrantTypeStats(baseUrl: string = "") {
  const [data, setData] = useState<GrantTypeData | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchStats = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/oauth/grant-type-stats`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setData(await res.json());
    } catch (e: any) { setError(e.message); }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { data, loading, error, fetchStats };
}
