import { useState, useCallback, useEffect } from "react";

/**
 * DEMO DATA — Tries real API first, falls back to empty demo data.
 * isDemoData flag indicates whether live or fallback data is shown.
 */

export interface GroupHealthMetric {
  group_name: string;
  member_count: number;
  member_activity_score: number;
  permission_freshness: number;
  status: string;
}

export interface CleanupRecommendation {
  action: string;
  group_name: string;
  reason: string;
  priority: string;
}

export interface IdentityGroupLifecycleData {
  groups_by_status: { active: number; dormant: number; empty: number; deprecated: number };
  auto_archive_after_days: number;
  group_health_metrics: GroupHealthMetric[];
  cleanup_recommendations: CleanupRecommendation[];
}

export function useIdentityGroupLifecycle() {
  const [data, setData] = useState<IdentityGroupLifecycleData | null>(null);
  const [isDemoData, setIsDemoData] = useState(true);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
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
      setData({
        groups_by_status: { active: 42, dormant: 8, empty: 5, deprecated: 3 },
        auto_archive_after_days: 90,
        group_health_metrics: [
          { group_name: "engineering-team", member_count: 28, member_activity_score: 0.92, permission_freshness: 12, status: "active" },
          { group_name: "qa-team", member_count: 8, member_activity_score: 0.75, permission_freshness: 45, status: "active" },
          { group_name: "contractors-2024", member_count: 3, member_activity_score: 0.15, permission_freshness: 120, status: "dormant" },
          { group_name: "legacy-admins", member_count: 0, member_activity_score: 0, permission_freshness: 365, status: "empty" },
          { group_name: "old-vendor-access", member_count: 2, member_activity_score: 0.05, permission_freshness: 200, status: "deprecated" },
        ],
        cleanup_recommendations: [
          { action: "Archive group", group_name: "legacy-admins", reason: "0 members, no activity in 365 days", priority: "high" },
          { action: "Review permissions", group_name: "old-vendor-access", reason: "Deprecated but still has 2 members", priority: "high" },
          { action: "Consider merge", group_name: "contractors-2024", reason: "Low activity, may overlap with active groups", priority: "medium" },
        ],
      });
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load data");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);
  return { data, loading, error, refresh: fetchData, isDemoData };
}
