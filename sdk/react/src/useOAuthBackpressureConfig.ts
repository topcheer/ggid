import { useState, useCallback } from "react";

export interface DegradationRule {
  metric: string;
  threshold: number;
  action: string;
}

export interface OAuthBackpressureConfig {
  per_client_fair_queueing: boolean;
  max_concurrent_token_requests: number;
  queue_overflow_action: "reject" | "defer";
  circuit_breaker_threshold: number;
  rate_limit_headers: boolean;
  graceful_degradation_rules: DegradationRule[];
}

export function useOAuthBackpressureConfig(baseUrl: string = "") {
  const [config, setConfig] = useState<OAuthBackpressureConfig | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchConfig = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/oauth-backpressure-config`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setConfig(await res.json());
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); }
    finally { setLoading(false); }
  }, [baseUrl]);

  const updateConfig = useCallback(async (patch: Partial<OAuthBackpressureConfig>) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/oauth-backpressure-config`, {
        method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(patch),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json(); setConfig(data); return data;
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); return null; }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { config, loading, error, fetchConfig, updateConfig };
}
