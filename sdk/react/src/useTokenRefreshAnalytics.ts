import { useState, useCallback, useEffect } from "react";

/**
 * DEMO DATA — Tries real API first, falls back to empty demo data.
 */

export interface RefreshByClient {
  client_id: string;
  client_name: string;
  refresh_count: number;
}

export interface RefreshFailure {
  error: string;
  count: number;
  description: string;
}

export interface TokenRefreshAnalyticsData {
  refresh_rate_per_hour: number[];
  avg_token_lifetime_minutes: number;
  refresh_success_rate: number;
  refresh_by_client: RefreshByClient[];
  refresh_failures: RefreshFailure[];
  rotation_churn_rate: number;
}

export function useTokenRefreshAnalytics() {
  const [data, setData] = useState<TokenRefreshAnalyticsData | null>(null);
  const [isDemoData, setIsDemoData] = useState(true);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      // Try real API first
      let res: Response | null = null;
      try { res = await fetch("/api/v1/data", { headers: { "Content-Type": "application/json" } }); } catch { res = null; }
      if (res?.ok) { const d = await res.json(); setData(d); setIsDemoData(false); return; }
      setIsDemoData(true);
      setData({
        refresh_rate_per_hour: Array.from({ length: 24 }, (_, i) => Math.round(50 + Math.sin(i / 3) * 30 + Math.random() * 20)),
        avg_token_lifetime_minutes: 45,
        refresh_success_rate: 96.8,
        refresh_by_client: [
          { client_id: "c1", client_name: "Web Dashboard", refresh_count: 4200 },
          { client_id: "c2", client_name: "Mobile App", refresh_count: 3100 },
          { client_id: "c3", client_name: "API Gateway", refresh_count: 1800 },
          { client_id: "c4", client_name: "CLI Tool", refresh_count: 520 },
        ],
        refresh_failures: [
          { error: "expired_refresh_token", count: 34, description: "Refresh token has expired" },
          { error: "invalid_grant", count: 12, description: "Invalid or revoked grant" },
          { error: "client_disabled", count: 3, description: "Client application disabled" },
        ],
        rotation_churn_rate: 12.5,
      });
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load data");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  return { data, loading, error, refresh: fetchData, isDemoData };
}
