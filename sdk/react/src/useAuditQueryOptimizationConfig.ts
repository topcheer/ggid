import { useState, useCallback } from "react";

export interface IndexConfig {
  table: string;
  columns: string[];
  type: "btree" | "hash" | "gin" | "brin";
}

export interface AutoVacuumConfig {
  enabled: boolean;
  threshold_pct: number;
  scale_factor: number;
}

export interface AuditQueryOptimizationConfig {
  partition_strategy: "daily" | "monthly";
  index_config: IndexConfig[];
  materialized_view_refresh_interval: number;
  cursor_pagination_size: number;
  slow_query_threshold_ms: number;
  auto_vacuum_config: AutoVacuumConfig;
}

export function useAuditQueryOptimizationConfig(baseUrl: string = "") {
  const [config, setConfig] = useState<AuditQueryOptimizationConfig | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchConfig = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/audit-query-optimization-config`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setConfig(await res.json());
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); }
    finally { setLoading(false); }
  }, [baseUrl]);

  const updateConfig = useCallback(async (patch: Partial<AuditQueryOptimizationConfig>) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/audit-query-optimization-config`, {
        method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(patch),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json(); setConfig(data); return data;
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); return null; }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { config, loading, error, fetchConfig, updateConfig };
}
