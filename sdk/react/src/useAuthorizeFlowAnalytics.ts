import { useState, useCallback } from "react";

export interface AuthorizeAnalytics {
  total_attempts: number;
  consent_rate: number;
  abandonment_at_step: { step: string; count: number; pct: number }[];
  avg_duration_ms: number;
  top_clients: { client_id: string; client_name: string; attempts: number; success_pct: number }[];
  pkce_adoption_pct: number;
  redirect_uri_errors: number;
}

export function useAuthorizeFlowAnalytics(baseUrl: string = "") {
  const [data, setData] = useState<AuthorizeAnalytics | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchAnalytics = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/oauth/authorize-flow-analytics`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setData(await res.json());
    } catch (e: any) { setError(e.message); }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { data, loading, error, fetchAnalytics };
}
