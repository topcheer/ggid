import { useState, useCallback } from "react";

export interface SecretInventoryEntry {
  name: string;
  source: string;
  last_rotated: string;
  age_days: number;
  status: "compliant" | "expiring" | "overdue";
}

export interface SecretSprawlPreventionConfig {
  scan_sources: string[];
  rotation_enforcement_days: number;
  vault_migration_status: string;
  ci_detection: boolean;
  runtime_validation: boolean;
  secret_inventory: SecretInventoryEntry[];
  violations_24h: number;
}

export function useSecretSprawlPreventionConfig(baseUrl: string = "") {
  const [config, setConfig] = useState<SecretSprawlPreventionConfig | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const fetchConfig = useCallback(async () => {
    setLoading(true); setError(null);
    try { const res = await fetch(`${baseUrl}/api/v1/settings/secret-sprawl-prevention-config`); if (!res.ok) throw new Error(`HTTP ${res.status}`); setConfig(await res.json()); }
    catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); } finally { setLoading(false); }
  }, [baseUrl]);
  const updateConfig = useCallback(async (patch: Partial<SecretSprawlPreventionConfig>) => {
    setLoading(true); setError(null);
    try { const res = await fetch(`${baseUrl}/api/v1/settings/secret-sprawl-prevention-config`, { method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(patch) }); if (!res.ok) throw new Error(`HTTP ${res.status}`); const data = await res.json(); setConfig(data); return data; }
    catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); return null; } finally { setLoading(false); }
  }, [baseUrl]);
  return { config, loading, error, fetchConfig, updateConfig };
}
