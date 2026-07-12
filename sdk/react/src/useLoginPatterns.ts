import { useState, useCallback } from "react";

export interface LoginPatternData {
  time_of_day: { hour: number; count: number }[];
  device_usage: { device: string; count: number }[];
  geo_distribution: { country: string; city: string; count: number }[];
  frequency_trend: { date: string; logins: number }[];
  anomalies: { type: string; description: string; severity: "low" | "medium" | "high" }[];
}

export function useLoginPatterns(baseUrl: string = "") {
  const [data, setData] = useState<LoginPatternData | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchPatterns = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/audit/login-patterns`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setData(await res.json());
    } catch (e: any) { setError(e.message); }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { data, loading, error, fetchPatterns };
}
