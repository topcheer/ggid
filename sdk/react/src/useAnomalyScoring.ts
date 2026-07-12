import { useState, useCallback } from "react";
export interface AnomalyData { signals: { name: string; weight: number }[]; thresholds: { low: number; medium: number; high: number; critical: number }; distribution: { bucket: string; count: number }[]; top_users: { username: string; score: number; top_signal: string; last_event: string }[]; model_stats: { accuracy: number; precision: number; recall: number; false_positive_rate: number }; }
export function useAnomalyScoring(baseUrl: string = "") {
  const [data, setData] = useState<AnomalyData | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const fetchData = useCallback(async () => { setLoading(true); setError(null); try { const res = await fetch(baseUrl + "/api/v1/auth/anomaly-scoring"); if (!res.ok) throw new Error("HTTP " + res.status); setData(await res.json()); } catch (e: any) { setError(e.message); } finally { setLoading(false); } }, [baseUrl]);
  return { data, loading, error, fetchData };
}
