import { useState, useCallback } from "react";

export interface TeamInsights {
  cohesion_score: number;
  collaboration_patterns: { team_a: string; team_b: string; frequency: number }[];
  silo_detection: { team: string; isolation_pct: number }[];
  cross_team_deps: { from: string; to: string; type: string }[];
  expertise_distribution: { team: string; skill: string; level: number }[];
  attrition_risk: { team: string; risk_level: "low" | "medium" | "high" }[];
}

export function useTeamInsights(baseUrl: string = "") {
  const [data, setData] = useState<TeamInsights | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchInsights = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/identity/team-insights`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setData(await res.json());
    } catch (e: any) { setError(e.message); }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { data, loading, error, fetchInsights };
}
