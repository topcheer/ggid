import { useState, useCallback } from "react";

export interface PerServiceSpan {
  service: string;
  sample_rate: number;
  max_spans: number;
}

export interface DistributedTracingConfig {
  otel_endpoint: string;
  sampling_rate: number;
  propagation_format: "W3C" | "Jaeger";
  per_service_span_config: PerServiceSpan[];
  trace_correlation_with_audit: boolean;
  backend: "Jaeger" | "Tempo";
}

export function useDistributedTracingConfig(baseUrl: string = "") {
  const [config, setConfig] = useState<DistributedTracingConfig | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const fetchConfig = useCallback(async () => {
    setLoading(true); setError(null);
    try { const res = await fetch(`${baseUrl}/api/v1/settings/distributed-tracing-config`); if (!res.ok) throw new Error(`HTTP ${res.status}`); setConfig(await res.json()); }
    catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); } finally { setLoading(false); }
  }, [baseUrl]);
  const updateConfig = useCallback(async (patch: Partial<DistributedTracingConfig>) => {
    setLoading(true); setError(null);
    try { const res = await fetch(`${baseUrl}/api/v1/settings/distributed-tracing-config`, { method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(patch) }); if (!res.ok) throw new Error(`HTTP ${res.status}`); const data = await res.json(); setConfig(data); return data; }
    catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); return null; } finally { setLoading(false); }
  }, [baseUrl]);
  return { config, loading, error, fetchConfig, updateConfig };
}
