import { useState, useCallback } from "react";

export interface PerTenantCanary {
  tenant_id: string;
  tenant_name: string;
  canary_enabled: boolean;
}

export interface PromotionCriteria {
  criterion: string;
  threshold: string;
  met: boolean;
}

export interface CanaryDeploymentConfig {
  canary_percentage: number;
  traffic_split_method: "header" | "weight" | "sticky";
  auto_rollback_on_error_rate: number;
  promotion_criteria: PromotionCriteria[];
  per_tenant_canary: PerTenantCanary[];
  monitoring_checkpoints: string[];
}

export function useCanaryDeploymentConfig(baseUrl: string = "") {
  const [config, setConfig] = useState<CanaryDeploymentConfig | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const fetchConfig = useCallback(async () => {
    setLoading(true); setError(null);
    try { const res = await fetch(`${baseUrl}/api/v1/settings/canary-deployment-config`); if (!res.ok) throw new Error(`HTTP ${res.status}`); setConfig(await res.json()); }
    catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); } finally { setLoading(false); }
  }, [baseUrl]);
  const updateConfig = useCallback(async (patch: Partial<CanaryDeploymentConfig>) => {
    setLoading(true); setError(null);
    try { const res = await fetch(`${baseUrl}/api/v1/settings/canary-deployment-config`, { method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(patch) }); if (!res.ok) throw new Error(`HTTP ${res.status}`); const data = await res.json(); setConfig(data); return data; }
    catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); return null; } finally { setLoading(false); }
  }, [baseUrl]);
  return { config, loading, error, fetchConfig, updateConfig };
}
