import { useState, useCallback, useEffect } from "react";

/**
 * DEMO DATA — Tries real API first, falls back to empty demo data.
 */

export interface StageTime {
  stage: string;
  avg_days: number;
}

export interface TransitionRule {
  rule: string;
  trigger: string;
}

export interface MonthlyTransition {
  month: string;
  count: number;
}

export interface UserLifecycleStatsData {
  stages: Record<string, number>;
  avg_time_per_stage: StageTime[];
  transition_rules: TransitionRule[];
  monthly_transitions: MonthlyTransition[];
}

export function useUserLifecycleStats() {
  const [data, setData] = useState<UserLifecycleStatsData | null>(null);
  const [isDemoData, setIsDemoData] = useState(true);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      // Try real API first
      let res: Response | null = null;
      try { res = await fetch("/api/v1/data", { headers: { "Content-Type": "application/json" } }); } catch { res = null; }
      if (res?.ok) { const d = await res.json(); setData(d); setIsDemoData(false); return; }
      setIsDemoData(true);
      setData({
        stages: { active: 4200, dormant: 380, suspended: 45, deactivated: 180, pending: 28 },
        avg_time_per_stage: [
          { stage: "pending", avg_days: 2 },
          { stage: "active", avg_days: 180 },
          { stage: "dormant", avg_days: 45 },
          { stage: "suspended", avg_days: 15 },
          { stage: "deactivated", avg_days: 90 },
        ],
        transition_rules: [
          { rule: "pending → active", trigger: "email_verified + first_login" },
          { rule: "active → dormant", trigger: "no_login_60d" },
          { rule: "dormant → suspended", trigger: "no_login_90d" },
          { rule: "suspended → deactivated", trigger: "no_login_180d" },
        ],
        monthly_transitions: [
          { month: "Jan", count: 45 }, { month: "Feb", count: 38 }, { month: "Mar", count: 52 }, { month: "Apr", count: 41 }, { month: "May", count: 33 }, { month: "Jun", count: 28 },
        ],
      });
    } catch (e) { setError(e instanceof Error ? e.message : "Failed"); }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);
  return { data, loading, error, refresh: fetchData, isDemoData };
}
