import { useState, useCallback } from "react";

export interface LogLevel {
  service: string;
  level: "debug" | "info" | "warn" | "error";
}

export interface RedactionRule {
  pattern: string;
  replacement: string;
}

export interface LogAggregationConfig {
  log_format: "JSON" | "structured";
  level_per_service: LogLevel[];
  correlation_id_enabled: boolean;
  sensitive_data_redaction_rules: RedactionRule[];
  log_routing: "Loki" | "ELK";
  retention_days: number;
  cost_optimization: boolean;
}

export function useLogAggregationConfig(baseUrl: string = "") {
  const [config, setConfig] = useState<LogAggregationConfig | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const fetchConfig = useCallback(async () => {
    setLoading(true); setError(null);
    try { const res = await fetch(`${baseUrl}/api/v1/settings/log-aggregation-config`); if (!res.ok) throw new Error(`HTTP ${res.status}`); setConfig(await res.json()); }
    catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); } finally { setLoading(false); }
  }, [baseUrl]);
  const updateConfig = useCallback(async (patch: Partial<LogAggregationConfig>) => {
    setLoading(true); setError(null);
    try { const res = await fetch(`${baseUrl}/api/v1/settings/log-aggregation-config`, { method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(patch) }); if (!res.ok) throw new Error(`HTTP ${res.status}`); const data = await res.json(); setConfig(data); return data; }
    catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); return null; } finally { setLoading(false); }
  }, [baseUrl]);
  return { config, loading, error, fetchConfig, updateConfig };
}
