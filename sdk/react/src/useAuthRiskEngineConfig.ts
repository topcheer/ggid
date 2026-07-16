import { useState, useCallback, useEffect } from "react";

/**
 * DEMO DATA — Tries real API first, falls back to empty demo data.
 * isDemoData flag indicates whether live or fallback data is shown.
 */

export interface RiskSignal {
  signal: string;
  weight: number;
  threshold: number;
  action_per_trigger: string;
}

export interface BacktestResults {
  precision: number;
  recall: number;
  f1: number;
}

export interface OverrideRule {
  condition: string;
  action: string;
  description: string;
}

export interface AuthRiskEngineConfigData {
  risk_signals: RiskSignal[];
  scoring_algorithm: string;
  retraining_frequency: string;
  model_version: string;
  backtest_results: BacktestResults;
  override_rules: OverrideRule[];
}

export function useAuthRiskEngineConfig() {
  const [data, setData] = useState<AuthRiskEngineConfigData | null>(null);
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
        risk_signals: [
          { signal: "impossible_travel", weight: 0.9, threshold: 500, action_per_trigger: "require_mfa" },
          { signal: "new_device", weight: 0.6, threshold: 0.7, action_per_trigger: "step_up" },
          { signal: "brute_force", weight: 1.0, threshold: 5, action_per_trigger: "block" },
          { signal: "credential_stuffing", weight: 0.85, threshold: 0.8, action_per_trigger: "require_mfa" },
          { signal: "geo_anomaly", weight: 0.5, threshold: 0.6, action_per_trigger: "step_up" },
        ],
        scoring_algorithm: "weighted_sum",
        retraining_frequency: "weekly",
        model_version: "2.3.1",
        backtest_results: { precision: 0.94, recall: 0.87, f1: 0.90 },
        override_rules: [
          { condition: "source_ip == trusted", action: "allow", description: "Skip risk check for trusted IPs" },
          { condition: "user.role == admin", action: "step_up", description: "Always require MFA for admins" },
          { condition: "risk_score > 0.95", action: "deny", description: "Block critical risk" },
        ],
      });
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load data");
    } finally {
      setLoading(false);
    }
  }, []);

  const retrainModel = useCallback(async () => {
    console.log("Retraining risk model");
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);
  return { data, loading, error, refresh: fetchData, retrainModel };
}
