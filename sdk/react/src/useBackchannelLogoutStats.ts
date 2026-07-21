import { useState, useCallback } from "react";

export interface BackchannelStats {
  total_requests: number;
  successful_pct: number;
  failed_count: number;
  top_failure_reasons: { reason: string; count: number }[];
  avg_latency_ms: number;
  by_idp_provider: { provider: string; requests: number; success_pct: number }[];
}

export function useBackchannelLogoutStats(baseUrl: string = "") {
  const [data, setData] = useState<BackchannelStats | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchStats = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(baseUrl + "/api/v1/auth/backchannel-logout-stats");
      if (!res.ok) throw new Error("HTTP " + res.status);
      setData(await res.json());
    } catch (e: any) { setError(e.message); }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { data, loading, error, fetchStats };
}
