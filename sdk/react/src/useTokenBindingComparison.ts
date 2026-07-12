import { useState, useCallback } from "react";

export interface ComparisonRow {
  method: string;
  security: number;
  deployment: number;
  performance: number;
  fallback: number;
}

export interface RecommendationEntry {
  use_case: string;
  recommended_method: string;
  rationale: string;
}

export interface BenchmarkResult {
  method: string;
  latency_ms: number;
  cpu_overhead_pct: number;
}

export interface PerClientMethod {
  client_id: string;
  client_name: string;
  method: string;
}

export interface TokenBindingComparison {
  comparison_table: ComparisonRow[];
  recommendation_matrix: RecommendationEntry[];
  benchmark_results: BenchmarkResult[];
  per_client_current_method: { client_id: string; client_name: string; method: string }[];
}

export function useTokenBindingComparison(baseUrl: string = "") {
  const [config, setConfig] = useState<TokenBindingComparison | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchConfig = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/token-binding-comparison`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setConfig(await res.json());
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); }
    finally { setLoading(false); }
  }, [baseUrl]);

  const updateConfig = useCallback(async (patch: Partial<TokenBindingComparison>) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/token-binding-comparison`, {
        method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(patch),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json(); setConfig(data); return data;
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); return null; }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { config, loading, error, fetchConfig, updateConfig };
}
