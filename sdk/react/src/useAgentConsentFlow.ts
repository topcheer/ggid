import { useState, useCallback, useEffect } from "react";

export interface PendingConsent {
  id: string;
  agent_name: string;
  requested_scopes: string[];
  resource: string;
  user: string;
  requested_at: string;
  expires_in: string;
  scope_justification: string;
}

export interface ConsentHistoryEntry {
  agent: string;
  user: string;
  scopes: string[];
  granted_at: string;
  revoked_at: string;
  status: string;
}

export interface AgentConsentFlowData {
  pending_consent_requests: PendingConsent[];
  consent_history: ConsentHistoryEntry[];
  active_agent_count: number;
  granular_scope_toggle: boolean;
  auto_expire_hours: number;
}

export function useAgentConsentFlow() {
  const [data, setData] = useState<AgentConsentFlowData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      await new Promise((r) => setTimeout(r, 400));
      setData({
        pending_consent_requests: [
          { id: "cr-001", agent_name: "CI/CD Bot", requested_scopes: ["deploy:staging", "read:config"], resource: "staging-cluster", user: "devops@ggid.dev", requested_at: "5m ago", expires_in: "55m", scope_justification: "Deploy v2.3.1 to staging environment" },
          { id: "cr-002", agent_name: "Data Pipeline", requested_scopes: ["read:users", "write:audit"], resource: "user-database", user: "data@ggid.dev", requested_at: "15m ago", expires_in: "45m", scope_justification: "Nightly ETL sync requires user data access" },
        ],
        consent_history: [
          { agent: "Monitoring Agent", user: "sre@ggid.dev", scopes: ["read:metrics", "read:health"], granted_at: "1h ago", revoked_at: "", status: "active" },
          { agent: "Security Scanner", user: "sec@ggid.dev", scopes: ["read:all", "scan:vulns"], granted_at: "2h ago", revoked_at: "", status: "active" },
          { agent: "Legacy Integration", user: "admin@ggid.dev", scopes: ["read:users"], granted_at: "5d ago", revoked_at: "2d ago", status: "revoked" },
          { agent: "CI/CD Bot", user: "devops@ggid.dev", scopes: ["deploy:staging"], granted_at: "6h ago", revoked_at: "", status: "active" },
        ],
        active_agent_count: 3,
        granular_scope_toggle: true,
        auto_expire_hours: 24,
      });
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load data");
    } finally {
      setLoading(false);
    }
  }, []);

  const approve = useCallback(async (id: string) => {
    console.log("Approving consent:", id);
  }, []);

  const deny = useCallback(async (id: string) => {
    console.log("Denying consent:", id);
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);
  return { data, loading, error, refresh: fetchData, approve, deny };
}
