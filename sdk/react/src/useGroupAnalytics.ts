import { useState, useCallback } from "react";

export interface GroupAnalytics {
  id: string;
  name: string;
  member_count: number;
  sub_groups: number;
  parent_groups: number;
  nested_depth: number;
  membership_trend_30d: { day: string; count: number }[];
  inactive_members: { user_id: string; username: string; last_active: string }[];
  role_assignments: number;
  access_review_status: "current" | "overdue" | "scheduled";
}

export function useGroupAnalytics(baseUrl: string = "") {
  const [groups, setGroups] = useState<GroupAnalytics[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchGroups = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(baseUrl + "/api/v1/org/group-analytics");
      if (!res.ok) throw new Error("HTTP " + res.status);
      const data = await res.json(); setGroups(data.groups || data || []);
    } catch (e: any) { setError(e.message); }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { groups, loading, error, fetchGroups };
}
