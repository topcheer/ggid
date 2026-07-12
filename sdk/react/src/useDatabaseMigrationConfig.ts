import { useState, useCallback } from "react";

export interface DatabaseMigrationConfig {
  migration_strategy: "expand_contract" | "big_bang" | "shadow";
  max_lock_duration_ms: number;
  batch_size: number;
  parallel_workers: number;
  backward_compat_window_days: number;
  rollback_timeout_seconds: number;
  dry_run: boolean;
}

export function useDatabaseMigrationConfig(baseUrl: string = "") {
  const [config, setConfig] = useState<DatabaseMigrationConfig | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const fetchConfig = useCallback(async () => {
    setLoading(true); setError(null);
    try { const res = await fetch(`${baseUrl}/api/v1/settings/database-migration-config`); if (!res.ok) throw new Error(`HTTP ${res.status}`); setConfig(await res.json()); }
    catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); } finally { setLoading(false); }
  }, [baseUrl]);
  const updateConfig = useCallback(async (patch: Partial<DatabaseMigrationConfig>) => {
    setLoading(true); setError(null);
    try { const res = await fetch(`${baseUrl}/api/v1/settings/database-migration-config`, { method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(patch) }); if (!res.ok) throw new Error(`HTTP ${res.status}`); const data = await res.json(); setConfig(data); return data; }
    catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); return null; } finally { setLoading(false); }
  }, [baseUrl]);
  return { config, loading, error, fetchConfig, updateConfig };
}
