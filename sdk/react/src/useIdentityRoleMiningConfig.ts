import { useState, useCallback, useEffect } from "react";

export interface MiningParameters {
  min_usage_threshold: number;
  co_occurrence_window_days: number;
  confidence_score_min: number;
}

export interface SuggestedRole {
  suggested_name: string;
  member_count: number;
  permission_count: number;
  confidence_score: number;
  key_permissions: string[];
}

export interface IdentityRoleMiningConfigData {
  mining_parameters: MiningParameters;
  auto_suggest_roles: boolean;
  similarity_algorithm: "jaccard" | "cosine" | "dice";
  last_mining_run: string | null;
  suggested_roles_review_queue: SuggestedRole[];
  applied_count: number;
}

export function useIdentityRoleMiningConfig() {
  const [data, setData] = useState<IdentityRoleMiningConfigData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      await new Promise((r) => setTimeout(r, 400));
      setData({
        mining_parameters: {
          min_usage_threshold: 10,
          co_occurrence_window_days: 30,
          confidence_score_min: 0.75,
        },
        auto_suggest_roles: true,
        similarity_algorithm: "jaccard",
        last_mining_run: "3h ago",
        suggested_roles_review_queue: [
          { suggested_name: "DevOps Engineer", member_count: 8, permission_count: 15, confidence_score: 0.92, key_permissions: ["deploy", "rollback", "logs:read", "metrics:read", "secrets:read"] },
          { suggested_name: "Read-Only Analyst", member_count: 12, permission_count: 6, confidence_score: 0.88, key_permissions: ["dashboard:view", "reports:read", "audit:read", "users:read"] },
          { suggested_name: "Onboarding Specialist", member_count: 4, permission_count: 10, confidence_score: 0.81, key_permissions: ["users:create", "roles:assign", "invitations:send"] },
        ],
        applied_count: 7,
      });
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load data");
    } finally {
      setLoading(false);
    }
  }, []);

  const runMining = useCallback(async () => {
    console.log("Running role mining");
  }, []);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  return { data, loading, error, refresh: fetchData, runMining };
}
