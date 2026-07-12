import { useState, useCallback } from "react";

export interface RiskData {
  user_id: string;
  username: string;
  risk_score: number;
  trend: number;
  factors: { key: string; label: string; score: number; max: number }[];
}

export function useRiskProfile(baseUrl: string = "") {
  const [data, setData] = useState<RiskData | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchRisk = useCallback(async (user: string) => {
    if (!user) return;
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/auth/risk-profile?user=${encodeURIComponent(user)}`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setData(await res.json());
    } catch (e: any) { setError(e.message); setData(null); }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { data, loading, error, fetchRisk };
}
