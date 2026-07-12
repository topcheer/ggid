import { useState, useCallback, useEffect } from "react";

export interface AlertRule {
  rule_name: string;
  condition: string;
  severity: string;
  channel: string;
}

export interface ActiveAlert {
  id: string;
  rule_name: string;
  message: string;
  severity: string;
  channel: string;
  triggered_at: string;
}

export interface SuppressionRule {
  dedup_key: string;
  suppress_minutes: number;
}

export interface EscalationStep {
  notify_after_minutes: number;
  escalate_to: string;
}

export interface AuditRealtimeAlertsData {
  alert_rules: AlertRule[];
  active_alerts: ActiveAlert[];
  alert_suppression_rules: SuppressionRule[];
  escalation_policy: EscalationStep[];
}

export function useAuditRealtimeAlerts() {
  const [data, setData] = useState<AuditRealtimeAlertsData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(false);
    setError(null);
    try {
      setData({
        alert_rules: [
          { rule_name: "Brute Force Detected", condition: "failed_logins > 10 in 60s", severity: "critical", channel: "pagerduty" },
          { rule_name: "Unusual Data Export", condition: "export_volume > 1GB in 1h", severity: "high", channel: "slack" },
          { rule_name: "After Hours Admin", condition: "admin_action AND hour NOT IN [8-18]", severity: "medium", channel: "email" },
          { rule_name: "Mass Token Revocation", condition: "revocations > 100 in 5m", severity: "high", channel: "webhook" },
        ],
        active_alerts: [
          { id: "alert-1", rule_name: "Brute Force Detected", message: "15 failed logins from IP 192.168.1.50 in 45s", severity: "critical", channel: "pagerduty", triggered_at: "2m ago" },
          { id: "alert-2", rule_name: "Unusual Data Export", message: "User eve.brown exported 2.3GB", severity: "high", channel: "slack", triggered_at: "8m ago" },
          { id: "alert-3", rule_name: "After Hours Admin", message: "Config change at 2:00 AM by admin.bob", severity: "medium", channel: "email", triggered_at: "15m ago" },
        ],
        alert_suppression_rules: [
          { dedup_key: "ip+rule", suppress_minutes: 5 },
          { dedup_key: "user+rule", suppress_minutes: 10 },
          { dedup_key: "global:critical", suppress_minutes: 1 },
        ],
        escalation_policy: [
          { notify_after_minutes: 0, escalate_to: "on-call engineer" },
          { notify_after_minutes: 15, escalate_to: "security team lead" },
          { notify_after_minutes: 30, escalate_to: "CISO" },
        ],
      });
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load data");
    }
  }, []);

  const acknowledgeAlert = useCallback(async (_id: string) => {
    console.log("Acknowledging alert:", _id);
  }, []);

  const testAlert = useCallback(async () => {
    console.log("Sending test alert");
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);
  return { data, loading, error, refresh: fetchData, acknowledgeAlert, testAlert };
}
