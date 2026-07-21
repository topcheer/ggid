import { useState, useCallback, useEffect } from "react";

/**
 * DEMO DATA — Tries real API first, falls back to empty demo data.
 */

export interface ActiveEmergency {
  policy: string;
  change_type: string;
  approved_by: string;
  effective_at: string;
  expires_at: string;
  time_remaining: string;
  time_remaining_pct: number;
}

export interface EmergencyHistoryEntry {
  id: string;
  policy: string;
  change_type: string;
  approved_by: string;
  timestamp: string;
  outcome: string;
}

export interface PolicyEmergencyChangesData {
  active_emergencies: ActiveEmergency[];
  post_incident_review_required: boolean;
  emergency_history: EmergencyHistoryEntry[];
}

export function usePolicyEmergencyChanges() {
  const [data, setData] = useState<PolicyEmergencyChangesData | null>(null);
  const [isDemoData, setIsDemoData] = useState(true);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      // Try real API first
      let res: Response | null = null;
      try { res = await fetch("/api/v1/data", { headers: { "Content-Type": "application/json" } }); } catch { res = null; }
      if (res?.ok) { const d = await res.json(); setData(d); setIsDemoData(false); return; }
      setIsDemoData(true);
      setData({
        active_emergencies: [
          { policy: "policy-prod-access", change_type: "bypass_mfa", approved_by: "CISO + CTO", effective_at: "10m ago", expires_at: "in 3h 50m", time_remaining: "3h 50m", time_remaining_pct: 95 },
        ],
        post_incident_review_required: true,
        emergency_history: [
          { id: "em-004", policy: "policy-admin-webauthn", change_type: "temporary_disable", approved_by: "Security Lead", timestamp: "2d ago", outcome: "reverted" },
          { id: "em-003", policy: "policy-ip-restrict", change_type: "add_ip_range", approved_by: "CISO + CTO", timestamp: "1w ago", outcome: "reverted" },
          { id: "em-002", policy: "policy-rate-limit", change_type: "increase_limit", approved_by: "On-call Admin", timestamp: "2w ago", outcome: "expired" },
          { id: "em-001", policy: "policy-geo-fence", change_type: "add_country", approved_by: "Security Lead", timestamp: "3w ago", outcome: "reverted" },
        ],
      });
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load data");
    } finally {
      setLoading(false);
    }
  }, []);

  const requestEmergency = useCallback(async () => {
    console.log("Requesting emergency change");
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);
  return { data, loading, error, refresh: fetchData, requestEmergency };
}
