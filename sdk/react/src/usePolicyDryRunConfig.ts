import { useState, useCallback } from "react";

export interface ContextValue {
  key: string;
  value: string;
}

export interface PolicyDryRunConfig {
  default_context_values: ContextValue[];
  max_simulation_subjects: number;
  cache_results_minutes: number;
  compare_against_current: boolean;
  auto_run_on_policy_change: boolean;
  results_retention_days: number;
}

export function usePolicyDryRunConfig(baseUrl: string = "") {
  const [config, setConfig] = useState<PolicyDryRunConfig | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchConfig = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/policy-dry-run-config`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setConfig(await res.json());
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); }
    finally { setLoading(false); }
  }, [baseUrl]);

  const updateConfig = useCallback(async (patch: Partial<PolicyDryRunConfig>) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/policy-dry-run-config`, {
        method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(patch),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json(); setConfig(data); return data;
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); return null; }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { config, loading, error, fetchConfig, updateConfig };
}
