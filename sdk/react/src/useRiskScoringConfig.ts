import { useState, useCallback } from "react";

export interface RiskConfig {
  weights: { geo_velocity: number; ip_reputation: number; device_familiarity: number; time_anomaly: number; failed_attempts: number };
  thresholds: { low: number; medium: number; high: number; critical: number };
  actions_per_level: { low: string; medium: string; high: string; critical: string };
  adaptive_mfa_trigger: boolean;
}

export function useRiskScoringConfig(baseUrl: string = "") {
  const [config, setConfig] = useState<RiskConfig | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchConfig = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(baseUrl + "/api/v1/auth/risk-scoring-config");
      if (!res.ok) throw new Error("HTTP " + res.status);
      setConfig(await res.json());
    } catch (e: any) { setError(e.message); }
    finally { setLoading(false); }
  }, [baseUrl]);

  const saveConfig = useCallback(async (cfg: RiskConfig) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(baseUrl + "/api/v1/auth/risk-scoring-config", { method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(cfg) });
      if (!res.ok) throw new Error("HTTP " + res.status);
      setConfig(cfg); return true;
    } catch (e: any) { setError(e.message); return false; }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { config, loading, error, fetchConfig, saveConfig };
}
