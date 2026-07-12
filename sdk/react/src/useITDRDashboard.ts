import { useState, useCallback, useEffect } from "react";

export interface ThreatDetection {
  id: string;
  type: string;
  severity: string;
  source: string;
  timestamp: string;
  affected_users: number;
  status: string;
  mitre_techniques: string[];
}

export interface DetectionRule {
  rule_name: string;
  technique: string;
  enabled: boolean;
  last_triggered: string;
}

export interface ResponsePlaybook {
  threat_type: string;
  steps_count: number;
  estimated_time: string;
  auto_execute: boolean;
}

export interface ITDRDashboardData {
  threat_detections: ThreatDetection[];
  detection_rules: DetectionRule[];
  response_playbooks: ResponsePlaybook[];
  auto_response_enabled: boolean;
}

export function useITDRDashboard() {
  const [data, setData] = useState<ITDRDashboardData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      await new Promise((r) => setTimeout(r, 400));
      setData({
        threat_detections: [
          { id: "td-001", type: "Credential Stuffing", severity: "critical", source: "auth-service", timestamp: "5m ago", affected_users: 47, status: "active", mitre_techniques: ["T1110.004"] },
          { id: "td-002", type: "Pass-the-Hash Attempt", severity: "high", source: "gateway", timestamp: "20m ago", affected_users: 3, status: "mitigated", mitre_techniques: ["T1550.002"] },
          { id: "td-003", type: "Golden Ticket Detection", severity: "critical", source: "kdc-monitor", timestamp: "1h ago", affected_users: 1, status: "resolved", mitre_techniques: ["T1558.001"] },
          { id: "td-004", type: "Anomalous LDAP Query", severity: "medium", source: "ldap-audit", timestamp: "2h ago", affected_users: 0, status: "resolved", mitre_techniques: ["T1018"] },
        ],
        detection_rules: [
          { rule_name: "Brute Force Detector", technique: "T1110", enabled: true, last_triggered: "5m ago" },
          { rule_name: "Impossible Travel", technique: "T1021", enabled: true, last_triggered: "30m ago" },
          { rule_name: "Token Theft Detector", technique: "T1528", enabled: true, last_triggered: "1h ago" },
          { rule_name: "Kerberoasting Detection", technique: "T1558", enabled: true, last_triggered: "3h ago" },
          { rule_name: "DCShadow Detection", technique: "T1207", enabled: false, last_triggered: "never" },
        ],
        response_playbooks: [
          { threat_type: "Credential Stuffing", steps_count: 6, estimated_time: "~5 min", auto_execute: true },
          { threat_type: "Token Theft", steps_count: 8, estimated_time: "~10 min", auto_execute: true },
          { threat_type: "Golden Ticket", steps_count: 12, estimated_time: "~30 min", auto_execute: false },
        ],
        auto_response_enabled: true,
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
