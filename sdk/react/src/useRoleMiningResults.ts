import { useState, useCallback, useEffect } from "react";

export interface UnusedPerm {
  permission: string;
  user: string;
  last_used_days: number;
}

export interface ConsolidationSuggestion {
  merge_target: string;
  roles_to_merge: string[];
  reduction_benefit: number;
}

export interface OverAssigned {
  user: string;
  role: string;
  excess_permissions: number;
}

export interface RoleMiningResultsData {
  creep_score: number;
  unused_permissions: UnusedPerm[];
  suggested_consolidation: ConsolidationSuggestion[];
  over_assigned: OverAssigned[];
}

export function useRoleMiningResults() {
  const [data, setData] = useState<RoleMiningResultsData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const fetchData = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      await new Promise((r) => setTimeout(r, 400));
      setData({
        creep_score: 62,
        unused_permissions: [
          { permission: "admin:billing:read", user: "user_js", last_used_days: 120 },
          { permission: "data:export:pii", user: "user_mk", last_used_days: 95 },
          { permission: "config:ssl:modify", user: "user_al", last_used_days: 200 },
        ],
        suggested_consolidation: [
          { merge_target: "developer-fullstack", roles_to_merge: ["dev-frontend", "dev-backend", "dev-infra"], reduction_benefit: 28 },
          { merge_target: "analyst-readonly", roles_to_merge: ["data-viewer", "report-reader"], reduction_benefit: 15 },
        ],
        over_assigned: [
          { user: "user_jd", role: "admin", excess_permissions: 8 },
          { user: "user_sm", role: "developer", excess_permissions: 3 },
        ],
      });
    } catch (e) { setError(e instanceof Error ? e.message : "Failed"); }
    finally { setLoading(false); }
  }, []);
  useEffect(() => { fetchData(); }, [fetchData]);
  return { data, loading, error, refresh: fetchData };
}
