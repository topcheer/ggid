import { useState, useCallback, useEffect } from "react";

export interface ScopeUsageEntry {
  scope_name: string;
  requested_count: number;
  granted_count: number;
  denied_count: number;
  deny_reasons: string[];
  avg_per_token: number;
}

export interface OAuthScopeAnalyticsData {
  scope_usage: ScopeUsageEntry[];
  scope_correlation: number[][];
  unused_scopes: string[];
}

export function useOAuthScopeAnalytics() {
  const [data, setData] = useState<OAuthScopeAnalyticsData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      await new Promise((r) => setTimeout(r, 400));
      setData({
        scope_usage: [
          { scope_name: "openid", requested_count: 45200, granted_count: 44980, denied_count: 220, deny_reasons: ["invalid_request"], avg_per_token: 1.0 },
          { scope_name: "profile", requested_count: 38100, granted_count: 37500, denied_count: 600, deny_reasons: ["insufficient_permissions", "consent_denied"], avg_per_token: 0.84 },
          { scope_name: "email", requested_count: 35000, granted_count: 34800, denied_count: 200, deny_reasons: ["consent_denied"], avg_per_token: 0.77 },
          { scope_name: "offline_access", requested_count: 12000, granted_count: 9500, denied_count: 2500, deny_reasons: ["not_allowed", "consent_denied", "policy_restricted"], avg_per_token: 0.26 },
          { scope_name: "admin:read", requested_count: 3200, granted_count: 1800, denied_count: 1400, deny_reasons: ["insufficient_permissions", "role_required"], avg_per_token: 0.07 },
          { scope_name: "admin:write", requested_count: 800, granted_count: 350, denied_count: 450, deny_reasons: ["insufficient_permissions", "mfa_required", "role_required"], avg_per_token: 0.02 },
        ],
        scope_correlation: [
          [1.0, 0.84, 0.77, 0.26, 0.07, 0.02],
          [0.84, 1.0, 0.92, 0.22, 0.05, 0.01],
          [0.77, 0.92, 1.0, 0.20, 0.03, 0.01],
          [0.26, 0.22, 0.20, 1.0, 0.15, 0.08],
          [0.07, 0.05, 0.03, 0.15, 1.0, 0.88],
          [0.02, 0.01, 0.01, 0.08, 0.88, 1.0],
        ],
        unused_scopes: ["legacy:read", "deprecated:export", "temp:migration"],
      });
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load data");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);
  return { data, loading, error, refresh: fetchData };
}
