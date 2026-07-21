import { useState, useCallback } from "react";

export interface AppIntegration {
  application_id: string;
  application_name: string;
  prediction_enabled: boolean;
  custom_interval: number;
}

export interface IdentityTokenPrefetchConfig {
  preemptive_refresh_threshold_pct: number;
  background_rotation_interval: number;
  client_prediction_model: "linear" | "exponential" | "ml";
  grace_period_seconds: number;
  offline_fallback_duration: number;
  per_app_integration: AppIntegration[];
}

export function useIdentityTokenPrefetchConfig(baseUrl: string = "") {
  const [config, setConfig] = useState<IdentityTokenPrefetchConfig | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchConfig = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/identity-token-prefetch-config`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setConfig(await res.json());
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); }
    finally { setLoading(false); }
  }, [baseUrl]);

  const updateConfig = useCallback(async (patch: Partial<IdentityTokenPrefetchConfig>) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/identity-token-prefetch-config`, {
        method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(patch),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json(); setConfig(data); return data;
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); return null; }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { config, loading, error, fetchConfig, updateConfig };
}
