import { useState, useCallback, useEffect } from "react";

export interface RegisteredAgent {
  agent_id: string;
  name: string;
  owner: string;
  status: string;
  created_at: string;
  last_active: string;
  permissions: string[];
  request_rate_per_min: number;
  rotation_due: boolean;
}

export interface CredentialRotationSchedule {
  interval_days: number;
  next_rotation: string;
}

export interface BehavioralAlert {
  agent_name: string;
  pattern: string;
  timestamp: string;
}

export interface AgentLifecycleData {
  registered_agents: RegisteredAgent[];
  credential_rotation_schedule: CredentialRotationSchedule;
  behavioral_alerts: BehavioralAlert[];
}

export function useAgentLifecycle() {
  const [data, setData] = useState<AgentLifecycleData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      await new Promise((r) => setTimeout(r, 400));
      setData({
        registered_agents: [
          { agent_id: "agent-001", name: "CI/CD Bot", owner: "devops@ggid.dev", status: "active", created_at: "30d ago", last_active: "2m ago", permissions: ["deploy:staging", "read:config"], request_rate_per_min: 45, rotation_due: false },
          { agent_id: "agent-002", name: "Monitoring Agent", owner: "sre@ggid.dev", status: "active", created_at: "60d ago", last_active: "30s ago", permissions: ["read:metrics", "read:health"], request_rate_per_min: 180, rotation_due: true },
          { agent_id: "agent-003", name: "Data Pipeline", owner: "data@ggid.dev", status: "suspended", created_at: "90d ago", last_active: "5h ago", permissions: ["read:users", "write:audit"], request_rate_per_min: 0, rotation_due: true },
          { agent_id: "agent-004", name: "Security Scanner", owner: "sec@ggid.dev", status: "active", created_at: "15d ago", last_active: "1h ago", permissions: ["read:all", "scan:vulns"], request_rate_per_min: 12, rotation_due: false },
          { agent_id: "agent-005", name: "Legacy Integration", owner: "admin@ggid.dev", status: "expired", created_at: "180d ago", last_active: "30d ago", permissions: ["read:users"], request_rate_per_min: 0, rotation_due: true },
        ],
        credential_rotation_schedule: { interval_days: 90, next_rotation: "in 15 days" },
        behavioral_alerts: [
          { agent_name: "Monitoring Agent", pattern: "Unusual API pattern: 3x normal request rate to /api/v1/users", timestamp: "15m ago" },
          { agent_name: "CI/CD Bot", pattern: "Accessed endpoint outside declared scope: /api/v1/admin/config", timestamp: "1h ago" },
        ],
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
