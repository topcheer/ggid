import { useState, useCallback } from "react";
export interface RiskFactor { factor: string; weight: number; current_value: number; contribution: number; }
export interface RiskData { composite_score: number; factors: RiskFactor[]; monte_carlo: { p50: number; p90: number; p99: number }; trend_30d: { date: string; score: number }[]; }
export function useRiskQuantification(baseUrl: string = "") {
  const [data, setData] = useState<RiskData | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const fetchData = useCallback(async () => { setLoading(true); setError(null); try { const res = await fetch(baseUrl + "/api/v1/audit/risk-quantification"); if (!res.ok) throw new Error("HTTP " + res.status); setData(await res.json()); } catch (e: any) { setError(e.message); } finally { setLoading(false); } }, [baseUrl]);
  return { data, loading, error, fetchData };
}
