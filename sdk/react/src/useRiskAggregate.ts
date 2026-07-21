import { useState, useCallback } from "react";

export interface HighRiskEntry {
  user_id: string;
  username: string;
  score: number;
  org: string;
  factors: string[];
}

export interface RiskData {
  avg_score: number;
  high_risk_count: number;
  trends_7d: number[];
  high_risk_users: HighRiskEntry[];
}

export function useRiskAggregate(baseUrl: string = "") {
  const [data, setData] = useState<RiskData | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchRisk = useCallback(async (view: "user" | "org" = "user") => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/audit/risk-aggregate?view=${view}`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setData(await res.json());
    } catch (e: any) { setError(e.message); setData(null); }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { data, loading, error, fetchRisk };
}
