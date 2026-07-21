import { useState, useCallback } from "react";

export interface BenchmarkResults {
  total_evaluations: number;
  avg_latency_ms: number;
  p99_latency_ms: number;
  cache_hit_rate_pct: number;
}

export interface PolicyEvaluationEngineConfig {
  rbac_fast_path: boolean;
  abac_cel_timeout_ms: number;
  cache_ttl_seconds: number;
  max_cache_entries: number;
  decision_tree_optimization: boolean;
  hot_path_threshold: number;
  benchmark_results: BenchmarkResults;
}

export function usePolicyEvaluationEngineConfig(baseUrl: string = "") {
  const [config, setConfig] = useState<PolicyEvaluationEngineConfig | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchConfig = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/policy-evaluation-engine-config`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setConfig(await res.json());
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); }
    finally { setLoading(false); }
  }, [baseUrl]);

  const updateConfig = useCallback(async (patch: Partial<PolicyEvaluationEngineConfig>) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/policy-evaluation-engine-config`, {
        method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(patch),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json(); setConfig(data); return data;
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); return null; }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { config, loading, error, fetchConfig, updateConfig };
}
