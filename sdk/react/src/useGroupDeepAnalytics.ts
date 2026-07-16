import { useState, useCallback, useEffect } from "react";

/**
 * DEMO DATA — Tries real API first, falls back to empty demo data.
 * isDemoData flag indicates whether live or fallback data is shown.
 */

export interface RoleDist {
  role: string;
  pct: number;
}

export interface GroupAnalytics {
  name: string;
  member_count: number;
  nesting_depth: number;
  permission_count: number;
  activity_score: number;
  heatmap: number[][];
  role_distribution: RoleDist[];
  anomalies: string[];
}

export interface GroupDeepAnalyticsData {
  groups: GroupAnalytics[];
}

export function useGroupDeepAnalytics() {
  const [data, setData] = useState<GroupDeepAnalyticsData | null>(null);
  const [isDemoData, setIsDemoData] = useState(true);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      // Try real API first
      let res: Response | null = null;
      try {
        res = await fetch("/api/v1/data", {
          headers: { "Content-Type": "application/json" },
        });
      } catch { res = null; }
      
      if (res?.ok) {
        const realData = await res.json();
        setData(realData);
        setIsDemoData(false);
        return;
      }
      
      // Fallback: empty demo data (no dangerous flags)
      setIsDemoData(true);
      const heatmap = Array.from({ length: 7 }, () => Array.from({ length: 24 }, () => Math.floor(Math.random() * 12)));
      setData({
        groups: [
          { name: "engineering", member_count: 85, nesting_depth: 3, permission_count: 42, activity_score: 82, heatmap, role_distribution: [{ role: "developer", pct: 55 }, { role: "senior_dev", pct: 25 }, { role: "lead", pct: 12 }, { role: "intern", pct: 8 }], anomalies: ["3 members active at 3AM (unusual hours)", "1 member accessed admin endpoint (not in role)"] },
          { name: "security-team", member_count: 12, nesting_depth: 2, permission_count: 68, activity_score: 91, heatmap, role_distribution: [{ role: "analyst", pct: 50 }, { role: "senior_analyst", pct: 33 }, { role: "admin", pct: 17 }], anomalies: ["High permission count (68) for 12 members"] },
        ],
      });
    } catch (e) { setError(e instanceof Error ? e.message : "Failed"); }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);
  return { data, loading, error, refresh: fetchData, isDemoData };
}
