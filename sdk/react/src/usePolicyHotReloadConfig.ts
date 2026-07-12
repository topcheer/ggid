import { useState, useCallback } from "react";

export interface PolicyHotReloadConfig {
  watch_enabled: boolean;
  atomic_swap: boolean;
  cache_invalidation_strategy: "all" | "lazy" | "versioned";
  version_check_interval_ms: number;
  rollback_on_error: boolean;
  max_reload_concurrency: number;
}

export function usePolicyHotReloadConfig(baseUrl: string = "") {
  const [config, setConfig] = useState<PolicyHotReloadConfig | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchConfig = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/policy-hot-reload-config`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setConfig(await res.json());
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); }
    finally { setLoading(false); }
  }, [baseUrl]);

  const updateConfig = useCallback(async (patch: Partial<PolicyHotReloadConfig>) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/policy-hot-reload-config`, {
        method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(patch),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json(); setConfig(data); return data;
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); return null; }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { config, loading, error, fetchConfig, updateConfig };
}
