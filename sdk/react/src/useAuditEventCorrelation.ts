import { useState, useCallback, useEffect } from "react";

/**
 * DEMO DATA — Tries real API first, falls back to empty demo data.
 * isDemoData flag indicates whether live or fallback data is shown.
 */

export interface CorrelatedIncident {
  id: string;
  title: string;
  description: string;
  severity: string;
  event_count: number;
  correlation_key: string;
  timestamp: string;
  events: string[];
}

export interface CorrelationRule {
  rule: string;
  window: string;
  min_events: number;
  action: string;
}

export interface AuditEventCorrelationData {
  engine_status: string;
  correlated_incidents: CorrelatedIncident[];
  correlation_rules: CorrelationRule[];
}

export function useAuditEventCorrelation() {
  const [data, setData] = useState<AuditEventCorrelationData | null>(null);
  const [isDemoData, setIsDemoData] = useState(true);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      // Try real API first
      let res: Response | null = null;
      try {
        res = await fetch("/api/v1/data", {
          headers: { "Content-Type": "application/json" },
        });
      } catch { res = null; }
      
      if (res?.ok) {
        const realData = await res.json();
        setData(realData);
        setIsDemoData(false);
        return;
      }
      
      // Fallback: empty demo data (no dangerous flags)
      setIsDemoData(true);
      setData({
        engine_status: "running",
        correlated_incidents: [
          { id: "ci-001", title: "Brute Force Pattern", description: "12 failed logins from 203.0.113.50 targeting 4 users", severity: "critical", event_count: 12, correlation_key: "source_ip", timestamp: "2h ago", events: ["auth.login.failed x8", "auth.login.failed x4"] },
          { id: "ci-002", title: "Token Reuse", description: "Refresh token used from 2 different IPs", severity: "high", event_count: 3, correlation_key: "token_id", timestamp: "5h ago", events: ["oauth.token.refresh", "oauth.token.refresh", "session.anomaly"] },
        ],
        correlation_rules: [
          { rule: "Multiple failed logins same IP", window: "5 min", min_events: 5, action: "Create incident" },
          { rule: "Token reuse from different IPs", window: "1 hour", min_events: 2, action: "Alert security" },
          { rule: "Privilege escalation chain", window: "30 min", min_events: 3, action: "Auto-block" },
          { rule: "After-hours admin access", window: "1 hour", min_events: 1, action: "Notify on-call" },
        ],
      });
    } catch (e) { setError(e instanceof Error ? e.message : "Failed"); }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);
  return { data, loading, error, refresh: fetchData, isDemoData };
}
