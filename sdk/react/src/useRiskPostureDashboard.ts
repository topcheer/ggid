import { useState, useCallback } from "react";
export interface RiskPosture { overall_score: number; categories: { category: string; score: number; max: number }[]; trending_risks: { name: string; trend: string }[]; mitigated_count: number; open_findings: { id: string; finding: string; severity: string; age_days: number; owner: string; status: string }[]; trend_30d: { date: string; score: number }[]; }
export function useRiskPostureDashboard(baseUrl: string = "") {
  const [data, setData] = useState<RiskPosture | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const fetchData = useCallback(async () => { setLoading(true); setError(null); try { const res = await fetch(baseUrl + "/api/v1/audit/risk-posture"); if (!res.ok) throw new Error("HTTP " + res.status); setData(await res.json()); } catch (e: any) { setError(e.message); } finally { setLoading(false); } }, [baseUrl]);
  return { data, loading, error, fetchData };
}
