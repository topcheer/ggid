import { useState, useCallback, useEffect } from "react";

export interface CategoryScore {
  category: string;
  score: number;
  delta: number;
}

export interface BenchmarkComparison {
  industry_avg: number;
  top_10_pct: number;
}

export interface Recommendation {
  recommendation: string;
  category: string;
  potential_gain: number;
}

export interface SecurityPostureScoringData {
  overall_score: number;
  by_category: CategoryScore[];
  benchmark_comparison: BenchmarkComparison;
  trend_30d: number[];
  improvement_recommendations: Recommendation[];
}

export function useSecurityPostureScoring() {
  const [data, setData] = useState<SecurityPostureScoringData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      await new Promise((r) => setTimeout(r, 400));
      setData({
        overall_score: 78,
        by_category: [
          { category: "identity", score: 85, delta: 3 },
          { category: "access", score: 82, delta: 2 },
          { category: "data", score: 70, delta: -1 },
          { category: "infra", score: 75, delta: 5 },
          { category: "compliance", score: 80, delta: 1 },
        ],
        benchmark_comparison: { industry_avg: 72, top_10_pct: 88 },
        trend_30d: [72, 73, 71, 74, 75, 73, 76, 75, 77, 76, 78, 77, 78],
        improvement_recommendations: [
          { recommendation: "Enable DPoP for all SPA clients", category: "access", potential_gain: 4 },
          { recommendation: "Implement data loss prevention policies", category: "data", potential_gain: 6 },
          { recommendation: "Enforce mTLS between all microservices", category: "infra", potential_gain: 3 },
          { recommendation: "Complete PCI-DSS QSA assessment", category: "compliance", potential_gain: 5 },
        ],
      });
    } catch (e) { setError(e instanceof Error ? e.message : "Failed"); }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);
  return { data, loading, error, refresh: fetchData };
}
