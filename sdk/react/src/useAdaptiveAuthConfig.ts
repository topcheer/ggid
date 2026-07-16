import { useState, useCallback, useEffect } from "react";

/**
 * DEMO DATA — Tries real API first, falls back to empty demo data.
 * isDemoData flag indicates whether live or fallback data is shown.
 */

export interface RiskThreshold {
  level: string;
  score_range: string;
  required_factor: string;
}

export interface SignalWeight {
  signal: string;
  weight: number;
}

export interface RoleOverride {
  role: string;
  min_factor: string;
}

export interface AdaptiveAuthConfigData {
  risk_thresholds: RiskThreshold[];
  signal_weights: SignalWeight[];
  step_up_triggers: string[];
  role_overrides: RoleOverride[];
}

export function useAdaptiveAuthConfig() {
  const [data, setData] = useState<AdaptiveAuthConfigData | null>(null);
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
        risk_thresholds: [
          { level: "low", score_range: "0-30", required_factor: "password" },
          { level: "medium", score_range: "31-60", required_factor: "password + OTP" },
          { level: "high", score_range: "61-85", required_factor: "password + WebAuthn" },
          { level: "critical", score_range: "86-100", required_factor: "deny" },
        ],
        signal_weights: [
          { signal: "geo_anomaly", weight: 30 },
          { signal: "device_trust", weight: 25 },
          { signal: "time_of_day", weight: 15 },
          { signal: "ip_reputation", weight: 20 },
          { signal: "frequency", weight: 10 },
        ],
        step_up_triggers: ["sensitive_action:delete_user", "policy_change:modify_rbac", "admin_access:panel", "data_export:bulk_pii"],
        role_overrides: [
          { role: "admin", min_factor: "webauthn" },
          { role: "developer", min_factor: "totp" },
          { role: "user", min_factor: "password" },
        ],
      });
    } catch (e) { setError(e instanceof Error ? e.message : "Failed"); }
    finally { setLoading(false); }
  }, []);
  useEffect(() => { fetchData(); }, [fetchData]);
  return { data, loading, error, refresh: fetchData, isDemoData };
}
