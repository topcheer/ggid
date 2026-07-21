import { useState, useCallback, useEffect } from "react";

/**
 * DEMO DATA — Tries real API first, falls back to empty demo data.
 * isDemoData flag indicates whether live or fallback data is shown.
 */

export interface GroupCard {
  name: string;
  member_count: number;
  sub_groups: number;
  nesting_depth: number;
}

export interface InactiveMember {
  user: string;
  group: string;
  last_active_days: number;
}

export interface OrphanedGroup {
  name: string;
  last_used: string;
}

export interface CleanupRecommendation {
  action: string;
  detail: string;
  priority: "high" | "medium" | "low";
}

export interface IdentityGroupMembershipAnalyticsData {
  group_cards: GroupCard[];
  membership_growth_30d: number[];
  inactive_members: InactiveMember[];
  orphaned_groups: OrphanedGroup[];
  shadow_permissions_detected: number;
  recommend_cleanup: CleanupRecommendation[];
}

export function useIdentityGroupMembershipAnalytics() {
  const [data, setData] = useState<IdentityGroupMembershipAnalyticsData | null>(null);
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
        res = await fetch("/api/v1/data", { headers: { "Content-Type": "application/json" } });
      } catch { res = null; }
      if (res?.ok) { const d = await res.json(); setData(d); setIsDemoData(false); return; }
      setIsDemoData(true);
      setData({
        group_cards: [
          { name: "Engineering", member_count: 124, sub_groups: 4, nesting_depth: 1 },
          { name: "Finance", member_count: 32, sub_groups: 2, nesting_depth: 2 },
          { name: "Security Team", member_count: 8, sub_groups: 0, nesting_depth: 1 },
          { name: "Sales", member_count: 56, sub_groups: 3, nesting_depth: 2 },
          { name: "Marketing", member_count: 28, sub_groups: 1, nesting_depth: 1 },
          { name: "DevOps", member_count: 15, sub_groups: 0, nesting_depth: 2 },
          { name: "Compliance", member_count: 6, sub_groups: 0, nesting_depth: 1 },
        ],
        membership_growth_30d: Array.from({ length: 30 }, (_, i) => 260 + Math.round(Math.sin(i / 5) * 15 + i * 1.5)),
        inactive_members: [
          { user: "old.user1", group: "Engineering", last_active_days: 120 },
          { user: "contractor.bob", group: "Sales", last_active_days: 95 },
          { user: "former.admin", group: "Security Team", last_active_days: 200 },
          { user: "intern.alice", group: "Marketing", last_active_days: 45 },
        ],
        orphaned_groups: [
          { name: "Legacy QA Team", last_used: "180 days ago" },
          { name: "Old Project Alpha", last_used: "365 days ago" },
        ],
        shadow_permissions_detected: 7,
        recommend_cleanup: [
          { action: "Remove inactive members", detail: "4 members inactive > 90 days across groups", priority: "high" },
          { action: "Delete orphaned groups", detail: "2 groups with no activity > 6 months", priority: "medium" },
          { action: "Review shadow permissions", detail: "7 permissions detected without group membership", priority: "high" },
          { action: "Flatten nested groups", detail: "Finance group has 2 levels of nesting", priority: "low" },
        ],
      });
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load data");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  return { data, loading, error, refresh: fetchData, isDemoData };
}
