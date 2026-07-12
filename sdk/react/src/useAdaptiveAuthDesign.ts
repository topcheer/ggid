import { useState, useCallback } from "react";

export interface SignalConfig {
  signal: string;
  source: string;
  latency_ms: number;
  weight: number;
}

export interface ThresholdConfig {
  low: number;
  medium: number;
  high: number;
}

export interface ABTestConfig {
  enabled: boolean;
  variant_a_pct: number;
  variant_b_label: string;
}

export interface AdaptiveAuthDesign {
  risk_scoring_model: string;
  signal_collection: SignalConfig[];
  threshold_tuning: ThresholdConfig;
  ml_vs_rule_based: "rule" | "ml" | "hybrid";
  a_b_test: ABTestConfig;
}

export function useAdaptiveAuthDesign(baseUrl: string = "") {
  const [config, setConfig] = useState<AdaptiveAuthDesign | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchConfig = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/adaptive-auth-design`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setConfig(await res.json());
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); }
    finally { setLoading(false); }
  }, [baseUrl]);

  const updateConfig = useCallback(async (patch: Partial<AdaptiveAuthDesign>) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/adaptive-auth-design`, {
        method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(patch),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json(); setConfig(data); return data;
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); return null; }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { config, loading, error, fetchConfig, updateConfig };
}
