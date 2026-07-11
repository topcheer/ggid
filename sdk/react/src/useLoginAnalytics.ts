import { useState, useCallback } from "react";

export interface LoginAnalytics {
  total_attempts: number;
  successful: number;
  failed: number;
  success_rate: number;
  avg_duration_ms: number;
  method_breakdown: { method: string; count: number; percentage: number }[];
  failure_reasons: { reason: string; count: number }[];
  daily_trend: { date: string; success: number; failure: number }[];
}

export function useLoginAnalytics(baseUrl: string = "") {
  const [data, setData] = useState<LoginAnalytics | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchAnalytics = useCallback(async (startDate: string, endDate: string) => {
    if (!startDate || !endDate) return;
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/auth/login-analytics?start=${startDate}&end=${endDate}`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const json: LoginAnalytics = await res.json();
      setData(json);
    } catch (e: any) {
      setError(e.message);
      setData(null);
    } finally {
      setLoading(false);
    }
  }, [baseUrl]);

  return { data, loading, error, fetchAnalytics };
}
