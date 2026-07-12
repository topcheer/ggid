import { useState, useCallback, useEffect } from "react";

export interface PolicyStat {
  policy: string;
  allow: number;
  deny: number;
}

export interface ResourceTypeStat {
  type: string;
  pct: number;
}

export interface DeniedAction {
  action: string;
  count: number;
}

export interface PolicyDecisionStatsData {
  allow_count: number;
  deny_count: number;
  avg_eval_time_ms: number;
  cache_hit_rate: number;
  by_policy: PolicyStat[];
  by_resource_type: ResourceTypeStat[];
  top_denied_actions: DeniedAction[];
}

export function usePolicyDecisionStats() {
  const [data, setData] = useState<PolicyDecisionStatsData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      await new Promise((r) => setTimeout(r, 400));
      setData({
        allow_count: 145200,
        deny_count: 8400,
        avg_eval_time_ms: 38,
        cache_hit_rate: 87,
        by_policy: [
          { policy: "admin-access", allow: 1200, deny: 45 },
          { policy: "api-access", allow: 89000, deny: 5200 },
          { policy: "data-access", allow: 45000, deny: 2800 },
          { policy: "service-account", allow: 10000, deny: 355 },
        ],
        by_resource_type: [
          { type: "API Endpoint", pct: 55 },
          { type: "Database", pct: 22 },
          { type: "Dashboard", pct: 15 },
          { type: "File Storage", pct: 8 },
        ],
        top_denied_actions: [
          { action: "admin:delete_user", count: 1200 },
          { action: "data:export_pii", count: 850 },
          { action: "config:modify", count: 420 },
          { action: "policy:override", count: 180 },
        ],
      });
    } catch (e) { setError(e instanceof Error ? e.message : "Failed"); }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);
  return { data, loading, error, refresh: fetchData };
}
