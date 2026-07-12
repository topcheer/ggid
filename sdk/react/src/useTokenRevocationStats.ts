import { useState, useCallback } from "react";

export interface RevocationStats {
  total_revocations: number;
  by_reason: { reason: string; count: number }[];
  by_client: { client_id: string; client_name: string; count: number }[];
  trend_30d: { day: string; count: number }[];
  peak_revocation_hour: number;
}

export function useTokenRevocationStats(baseUrl: string = "") {
  const [data, setData] = useState<RevocationStats | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchStats = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(baseUrl + "/api/v1/oauth/token-revocation-stats");
      if (!res.ok) throw new Error("HTTP " + res.status);
      setData(await res.json());
    } catch (e: any) { setError(e.message); }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { data, loading, error, fetchStats };
}
