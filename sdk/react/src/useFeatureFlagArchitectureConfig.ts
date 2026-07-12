import { useState, useCallback } from "react";

export interface KillSwitch {
  name: string;
  description: string;
  enabled: boolean;
}

export interface PerTenantFlag {
  tenant_id: string;
  tenant_name: string;
  flags: Record<string, boolean>;
}

export interface FeatureFlagArchitectureConfig {
  flag_types: string[];
  evaluation_engine: "local" | "remote" | "hybrid";
  rollout_strategies: string[];
  kill_switches: KillSwitch[];
  per_tenant_flags: PerTenantFlag[];
  a_b_test_config: { enabled: boolean; variants: string[] };
}

export function useFeatureFlagArchitectureConfig(baseUrl: string = "") {
  const [config, setConfig] = useState<FeatureFlagArchitectureConfig | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const fetchConfig = useCallback(async () => {
    setLoading(true); setError(null);
    try { const res = await fetch(`${baseUrl}/api/v1/settings/feature-flag-architecture-config`); if (!res.ok) throw new Error(`HTTP ${res.status}`); setConfig(await res.json()); }
    catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); } finally { setLoading(false); }
  }, [baseUrl]);
  const updateConfig = useCallback(async (patch: Partial<FeatureFlagArchitectureConfig>) => {
    setLoading(true); setError(null);
    try { const res = await fetch(`${baseUrl}/api/v1/settings/feature-flag-architecture-config`, { method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(patch) }); if (!res.ok) throw new Error(`HTTP ${res.status}`); const data = await res.json(); setConfig(data); return data; }
    catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); return null; } finally { setLoading(false); }
  }, [baseUrl]);
  return { config, loading, error, fetchConfig, updateConfig };
}
