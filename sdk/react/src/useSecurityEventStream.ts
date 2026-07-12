import { useState, useCallback, useEffect } from "react";

export interface SecurityEvent {
  id: string;
  type: string;
  severity: string;
  message: string;
  source: string;
  timestamp: string;
  affected_entities: string[];
  correlation_id: string;
  raw_data: string;
}

export interface SecurityEventStreamData {
  events: SecurityEvent[];
}

export function useSecurityEventStream() {
  const [data, setData] = useState<SecurityEventStreamData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      await new Promise((r) => setTimeout(r, 400));
      setData({
        events: [
          { id: "ev-001", type: "auth.brute_force", severity: "critical", message: "12 failed logins from 203.0.113.50 in 60s", source: "auth-service", timestamp: "2s ago", affected_entities: ["alice@corp.com", "bob@corp.com"], correlation_id: "corr-789", raw_data: '{"src_ip":"203.0.113.50","attempts":12,"window_sec":60}' },
          { id: "ev-002", type: "policy.violation", severity: "high", message: "User accessed resource outside policy scope", source: "policy-service", timestamp: "15s ago", affected_entities: ["svc.legacy"], correlation_id: "", raw_data: '{"user":"svc.legacy","resource":"admin-panel","action":"GET"}' },
          { id: "ev-003", type: "oauth.token_reuse", severity: "high", message: "Refresh token used from new IP", source: "oauth-service", timestamp: "45s ago", affected_entities: ["diana@corp.com"], correlation_id: "corr-456", raw_data: '{"token_id":"rt_abc123","old_ip":"10.0.0.5","new_ip":"198.51.100.22"}' },
          { id: "ev-004", type: "session.anomaly", severity: "medium", message: "Concurrent sessions from different countries", source: "identity-service", timestamp: "1m ago", affected_entities: ["admin@corp.com"], correlation_id: "", raw_data: '{"user":"admin","locations":["US","NL"]}' },
          { id: "ev-005", type: "audit.gap", severity: "low", message: "Missing audit event for admin action", source: "audit-service", timestamp: "2m ago", affected_entities: [], correlation_id: "", raw_data: '{}' },
        ],
      });
    } catch (e) { setError(e instanceof Error ? e.message : "Failed"); }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);
  return { data, loading, error, refresh: fetchData };
}
