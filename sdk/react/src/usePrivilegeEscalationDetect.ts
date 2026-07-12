import { useState, useCallback, useEffect } from "react";

export interface EscalationEvent {
  id: string;
  user: string;
  from_role: string;
  to_role: string;
  method: string;
  patterns: string[];
  confidence_score: number;
  action_taken: string;
  timestamp: string;
}

export interface RecommendedAction {
  action: string;
  reason: string;
  priority: string;
}

export interface PrivilegeEscalationDetectData {
  detected_events: EscalationEvent[];
  recommended_actions: RecommendedAction[];
}

export function usePrivilegeEscalationDetect() {
  const [data, setData] = useState<PrivilegeEscalationDetectData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      await new Promise((r) => setTimeout(r, 400));
      setData({
        detected_events: [
          { id: "pe-001", user: "temp.admin@ggid.dev", from_role: "viewer", to_role: "admin", method: "API misuse", patterns: ["mass_grant", "bypass_workflow"], confidence_score: 0.95, action_taken: "blocked", timestamp: "10m ago" },
          { id: "pe-002", user: "contractor.ext@ggid.dev", from_role: "developer", to_role: "superadmin", method: "policy override", patterns: ["unusual_time", "bypass_workflow"], confidence_score: 0.88, action_taken: "reverted", timestamp: "1h ago" },
          { id: "pe-003", user: "user.service@ggid.dev", from_role: "reader", to_role: "writer", method: "token replay", patterns: ["mass_grant"], confidence_score: 0.72, action_taken: "flagged", timestamp: "3h ago" },
        ],
        recommended_actions: [
          { action: "Review admin access grants", reason: "3 events detected with mass_grant pattern", priority: "critical" },
          { action: "Enforce SoD for superadmin", reason: "Bypass workflow detected", priority: "high" },
          { action: "Enable step-up auth for role changes", reason: "Token replay method indicates compromised credentials", priority: "high" },
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
