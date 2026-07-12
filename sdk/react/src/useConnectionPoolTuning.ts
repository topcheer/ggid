import { useState, useCallback } from "react";

export interface PoolConfig {
  service: string;
  type: "PG" | "Redis" | "gRPC" | "HTTP";
  max: number;
  min: number;
  idle: number;
  current_utilization: number;
  leak_count: number;
}

export interface PoolBenchmark {
  throughput_rps: number;
  avg_latency_ms: number;
}

export interface ConnectionPoolTuning {
  pool_configs: PoolConfig[];
  sizing_recommendation: string;
  leak_detection: boolean;
  benchmark_results: PoolBenchmark;
}

export function useConnectionPoolTuning(baseUrl: string = "") {
  const [config, setConfig] = useState<ConnectionPoolTuning | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const fetchConfig = useCallback(async () => {
    setLoading(true); setError(null);
    try { const res = await fetch(`${baseUrl}/api/v1/settings/connection-pool-tuning`); if (!res.ok) throw new Error(`HTTP ${res.status}`); setConfig(await res.json()); }
    catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); } finally { setLoading(false); }
  }, [baseUrl]);
  const updateConfig = useCallback(async (patch: Partial<ConnectionPoolTuning>) => {
    setLoading(true); setError(null);
    try { const res = await fetch(`${baseUrl}/api/v1/settings/connection-pool-tuning`, { method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(patch) }); if (!res.ok) throw new Error(`HTTP ${res.status}`); const data = await res.json(); setConfig(data); return data; }
    catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); return null; } finally { setLoading(false); }
  }, [baseUrl]);
  return { config, loading, error, fetchConfig, updateConfig };
}
