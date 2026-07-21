import { useState, useCallback, useEffect } from "react";

/**
 * DEMO DATA — Tries real API first, falls back to empty demo data.
 * isDemoData flag indicates whether live or fallback data is shown.
 */

export interface RiskThresholdEntry {
  risk_level: string;
  required_factors: string[];
}

export interface SignalWeight {
  signal: string;
  weight: number;
}

export interface StepUpTrigger {
  action: string;
  required_level: string;
}

export interface RoleOverride {
  role: string;
  min_auth_level: string;
  max_session_minutes: number;
}

export interface AuthAdaptiveAuthFlowData {
  risk_threshold_matrix: RiskThresholdEntry[];
  signal_weights: SignalWeight[];
  step_up_triggers: StepUpTrigger[];
  override_per_role: RoleOverride[];
}

export function useAuthAdaptiveAuthFlow() {
  const [data, setData] = useState<AuthAdaptiveAuthFlowData | null>(null);
  const [isDemoData, setIsDemoData] = useState(true);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
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
        risk_threshold_matrix: [
          { risk_level: "low", required_factors: ["password"] },
          { risk_level: "medium", required_factors: ["password", "otp"] },
          { risk_level: "high", required_factors: ["password", "webauthn"] },
          { risk_level: "critical", required_factors: ["deny"] },
        ],
        signal_weights: [
          { signal: "geo", weight: 0.25 },
          { signal: "device", weight: 0.20 },
          { signal: "time", weight: 0.15 },
          { signal: "ip", weight: 0.25 },
          { signal: "frequency", weight: 0.15 },
        ],
        step_up_triggers: [
          { action: "sensitive_action", required_level: "medium" },
          { action: "policy_change", required_level: "high" },
          { action: "admin_access", required_level: "high" },
          { action: "data_export", required_level: "medium" },
        ],
        override_per_role: [
          { role: "Admin", min_auth_level: "high", max_session_minutes: 60 },
          { role: "Developer", min_auth_level: "medium", max_session_minutes: 480 },
          { role: "User", min_auth_level: "low", max_session_minutes: 480 },
        ],
      });
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load data");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);
  return { data, loading, error, refresh: fetchData, isDemoData };
}
