import { useState, useCallback, useEffect } from "react";

export interface PermissionDelta {
  user: string;
  added_perms: number;
  removed_perms: number;
  unchanged: number;
  risk_score_change: number;
}

export interface ComparisonItem {
  metric: string;
  value: string;
}

export interface PolicyAnalysis {
  policy_id: string;
  policy_name: string;
  affected_users_count: number;
  avg_risk_score_change: number;
  high_risk_users: number;
  permission_delta: PermissionDelta[];
  before: ComparisonItem[];
  after: ComparisonItem[];
  timeline_projection: number[];
}

export interface PolicyImpactAnalysisData {
  analyses: PolicyAnalysis[];
}

export function usePolicyImpactAnalysis() {
  const [data, setData] = useState<PolicyImpactAnalysisData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      await new Promise((r) => setTimeout(r, 400));
      setData({
        analyses: [
          {
            policy_id: "pol-001",
            policy_name: "Restrict Admin API Access",
            affected_users_count: 42,
            avg_risk_score_change: -12.5,
            high_risk_users: 3,
            permission_delta: [
              { user: "alice.chen", added_perms: 0, removed_perms: 5, unchanged: 12, risk_score_change: -18.0 },
              { user: "bob.martinez", added_perms: 0, removed_perms: 3, unchanged: 8, risk_score_change: -10.5 },
              { user: "carol.jones", added_perms: 2, removed_perms: 0, unchanged: 10, risk_score_change: 8.0 },
              { user: "dave.wilson", added_perms: 0, removed_perms: 4, unchanged: 6, risk_score_change: -15.0 },
              { user: "eve.brown", added_perms: 1, removed_perms: 1, unchanged: 14, risk_score_change: 2.5 },
              { user: "frank.lee", added_perms: 0, removed_perms: 6, unchanged: 9, risk_score_change: -22.0 },
              { user: "grace.kim", added_perms: 3, removed_perms: 0, unchanged: 5, risk_score_change: 15.5 },
              { user: "henry.chen", added_perms: 0, removed_perms: 2, unchanged: 20, risk_score_change: -5.0 },
            ],
            before: [
              { metric: "Total Permissions", value: "420" },
              { metric: "Admin Scopes", value: "38" },
              { metric: "Avg Risk Score", value: "62.3" },
              { metric: "Compliance Gaps", value: "7" },
            ],
            after: [
              { metric: "Total Permissions", value: "398" },
              { metric: "Admin Scopes", value: "24" },
              { metric: "Avg Risk Score", value: "49.8" },
              { metric: "Compliance Gaps", value: "3" },
            ],
            timeline_projection: [62, 58, 55, 53, 51, 50, 49, 49, 48, 49, 48, 48, 49, 50],
          },
          {
            policy_id: "pol-002",
            policy_name: "Add MFA for Finance Dept",
            affected_users_count: 28,
            avg_risk_score_change: -25.0,
            high_risk_users: 0,
            permission_delta: [
              { user: "fin.alice", added_perms: 1, removed_perms: 0, unchanged: 10, risk_score_change: -28.0 },
              { user: "fin.bob", added_perms: 1, removed_perms: 0, unchanged: 8, risk_score_change: -22.0 },
              { user: "fin.carol", added_perms: 1, removed_perms: 0, unchanged: 12, risk_score_change: -30.0 },
            ],
            before: [
              { metric: "MFA Coverage", value: "60%" },
              { metric: "Avg Risk Score", value: "55.0" },
              { metric: "Compliance Gaps", value: "4" },
            ],
            after: [
              { metric: "MFA Coverage", value: "100%" },
              { metric: "Avg Risk Score", value: "30.0" },
              { metric: "Compliance Gaps", value: "0" },
            ],
            timeline_projection: [55, 48, 42, 38, 35, 32, 30, 30, 29, 29, 30, 30, 31, 31],
          },
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

  return { data, loading, error, refresh: fetchData };
}
