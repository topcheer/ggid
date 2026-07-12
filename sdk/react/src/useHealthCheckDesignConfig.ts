import { useState, useCallback } from "react";

export interface DependencyCheck {
  name: string;
  status: "healthy" | "degraded" | "down";
  latency_ms: number;
}

export interface DegradationRule {
  condition: string;
  action: string;
}

export interface HealthCheckDesignConfig {
  check_types: string[];
  dependency_checks: DependencyCheck[];
  degradation_rules: DegradationRule[];
  circuit_breaker_integration: boolean;
  auto_healing: boolean;
  lb_integration: boolean;
}

export function useHealthCheckDesignConfig(baseUrl: string = "") {
  const [config, setConfig] = useState<HealthCheckDesignConfig | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const fetchConfig = useCallback(async () => {
    setLoading(true); setError(null);
    try { const res = await fetch(`${baseUrl}/api/v1/settings/health-check-design-config`); if (!res.ok) throw new Error(`HTTP ${res.status}`); setConfig(await res.json()); }
    catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); } finally { setLoading(false); }
  }, [baseUrl]);
  const updateConfig = useCallback(async (patch: Partial<HealthCheckDesignConfig>) => {
    setLoading(true); setError(null);
    try { const res = await fetch(`${baseUrl}/api/v1/settings/health-check-design-config`, { method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(patch) }); if (!res.ok) throw new Error(`HTTP ${res.status}`); const data = await res.json(); setConfig(data); return data; }
    catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); return null; } finally { setLoading(false); }
  }, [baseUrl]);
  return { config, loading, error, fetchConfig, updateConfig };
}
