import { useState, useCallback, useEffect } from "react";

export interface Environment {
  name: string;
  version: string;
  health: "healthy" | "degraded" | "down";
  last_deploy: string;
  active_clients: number;
}

export interface ConfigDiffEntry {
  field: string;
  change_type: "added" | "removed" | "modified";
  old_value: string;
  new_value: string;
}

export interface DeploymentRecord {
  id: string;
  environment: string;
  version: string;
  status: "success" | "failed" | "in_progress";
  deployed_by: string;
  timestamp: string;
  rollback_available: boolean;
}

export interface HealthCheck {
  check: string;
  status: "pass" | "warn" | "fail";
  latency_ms: number;
}

export interface OAuthClientDeploymentData {
  environments: Environment[];
  config_diff: ConfigDiffEntry[];
  deployment_history: DeploymentRecord[];
  health_checks: HealthCheck[];
}

export function useOAuthClientDeployment() {
  const [data, setData] = useState<OAuthClientDeploymentData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      await new Promise((r) => setTimeout(r, 400));
      setData({
        environments: [
          { name: "dev", version: "v2.4.1-beta", health: "healthy", last_deploy: "2h ago", active_clients: 8 },
          { name: "staging", version: "v2.4.0", health: "healthy", last_deploy: "1d ago", active_clients: 12 },
          { name: "prod", version: "v2.3.9", health: "healthy", last_deploy: "3d ago", active_clients: 28 },
        ],
        config_diff: [
          { field: "redirect_uris", change_type: "added", old_value: "", new_value: "https://app2.example.com/callback" },
          { field: "grant_types", change_type: "modified", old_value: "[authorization_code]", new_value: "[authorization_code, refresh_token]" },
          { field: "token_endpoint_auth_method", change_type: "modified", old_value: "client_secret_post", new_value: "client_secret_basic" },
          { field: "scope", change_type: "modified", old_value: "openid profile", new_value: "openid profile email" },
        ],
        deployment_history: [
          { id: "dep-1", environment: "dev", version: "v2.4.1-beta", status: "success", deployed_by: "alice", timestamp: "2h ago", rollback_available: true },
          { id: "dep-2", environment: "staging", version: "v2.4.0", status: "success", deployed_by: "ci-bot", timestamp: "1d ago", rollback_available: true },
          { id: "dep-3", environment: "prod", version: "v2.3.9", status: "success", deployed_by: "bob", timestamp: "3d ago", rollback_available: true },
          { id: "dep-4", environment: "prod", version: "v2.3.8", status: "success", deployed_by: "bob", timestamp: "7d ago", rollback_available: false },
          { id: "dep-5", environment: "staging", version: "v2.3.9-beta", status: "failed", deployed_by: "ci-bot", timestamp: "8d ago", rollback_available: false },
        ],
        health_checks: [
          { check: "Token Endpoint", status: "pass", latency_ms: 45 },
          { check: "Authorize Endpoint", status: "pass", latency_ms: 32 },
          { check: "UserInfo Endpoint", status: "pass", latency_ms: 28 },
          { check: "JWKS Endpoint", status: "pass", latency_ms: 12 },
          { check: "Revocation Endpoint", status: "warn", latency_ms: 120 },
          { check: "Introspection Endpoint", status: "pass", latency_ms: 55 },
        ],
      });
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load data");
    } finally {
      setLoading(false);
    }
  }, []);

  const promote = useCallback(async (env: string) => {
    console.log("Promoting from:", env);
  }, []);

  const rollback = useCallback(async (depId: string) => {
    console.log("Rolling back deployment:", depId);
  }, []);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  return { data, loading, error, refresh: fetchData, promote, rollback };
}
