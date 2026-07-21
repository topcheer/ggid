import { useState, useCallback, useEffect } from "react";

/**
 * DEMO DATA — Tries real API first, falls back to empty demo data.
 */

export interface ProviderStat {
  name: string;
  status: string;
  user_count: number;
  login_count_30d: number;
  success_rate: number;
  avg_latency_ms: number;
  new_users_30d: number;
}

export interface ProviderError {
  error: string;
  provider: string;
  count: number;
}

export interface SocialProviderStatsData {
  providers: ProviderStat[];
  top_errors: ProviderError[];
}

export function useSocialProviderStats() {
  const [data, setData] = useState<SocialProviderStatsData | null>(null);
  const [isDemoData, setIsDemoData] = useState(true);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      // Try real API first
      let res: Response | null = null;
      try { res = await fetch("/api/v1/data", { headers: { "Content-Type": "application/json" } }); } catch { res = null; }
      if (res?.ok) { const d = await res.json(); setData(d); setIsDemoData(false); return; }
      setIsDemoData(true);
      setData({
        providers: [
          { name: "Google", status: "active", user_count: 3200, login_count_30d: 18500, success_rate: 99.2, avg_latency_ms: 120, new_users_30d: 142 },
          { name: "GitHub", status: "active", user_count: 1800, login_count_30d: 9200, success_rate: 98.5, avg_latency_ms: 180, new_users_30d: 78 },
          { name: "Microsoft", status: "active", user_count: 950, login_count_30d: 4100, success_rate: 97.8, avg_latency_ms: 210, new_users_30d: 35 },
          { name: "Apple", status: "active", user_count: 420, login_count_30d: 1800, success_rate: 96.5, avg_latency_ms: 250, new_users_30d: 22 },
          { name: "Slack", status: "active", user_count: 680, login_count_30d: 3200, success_rate: 98.1, avg_latency_ms: 160, new_users_30d: 15 },
        ],
        top_errors: [
          { error: "invalid_grant", provider: "Microsoft", count: 89 },
          { error: "state_mismatch", provider: "Apple", count: 45 },
          { error: "user_cancelled", provider: "Google", count: 32 },
        ],
      });
    } catch (e) { setError(e instanceof Error ? e.message : "Failed"); }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);
  return { data, loading, error, refresh: fetchData, isDemoData };
}
