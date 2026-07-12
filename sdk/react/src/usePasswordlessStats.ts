import { useState, useCallback } from "react";

export interface PasswordlessStats {
  method_distribution: { method: string; count: number }[];
  success_rate: number;
  avg_completion_time_ms: number;
  abandonment_rate: number;
  by_device_type: { device: string; attempts: number; success_pct: number }[];
}

export function usePasswordlessStats(baseUrl: string = "") {
  const [data, setData] = useState<PasswordlessStats | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchStats = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/auth/passwordless-stats`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setData(await res.json());
    } catch (e: any) { setError(e.message); }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { data, loading, error, fetchStats };
}
