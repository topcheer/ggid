import { useState, useCallback } from "react";

export interface OptimizationData {
  user_id: string;
  username: string;
  optimization_score: number;
  redundant_roles: { role: string; overlaps_with: string; overlap_pct: number }[];
  unused_paths: { path: string; last_accessed: string }[];
  suggestions: { action: string; impact: string; roles_affected: string[] }[];
}

export function useAccessOptimization(baseUrl: string = "") {
  const [data, setData] = useState<OptimizationData | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchOptimization = useCallback(async (user: string) => {
    if (!user) return;
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/policy/access-optimization?user=${encodeURIComponent(user)}`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const json: OptimizationData = await res.json();
      setData(json);
    } catch (e: any) {
      setError(e.message);
      setData(null);
    } finally {
      setLoading(false);
    }
  }, [baseUrl]);

  return { data, loading, error, fetchOptimization };
}
