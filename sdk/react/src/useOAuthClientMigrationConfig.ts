import { useState, useCallback } from "react";

export interface MappingPreview {
  source_field: string;
  target_field: string;
  transform: string;
}

export interface ValidationReport {
  status: "pass" | "warn" | "fail";
  total_clients: number;
  mapped: number;
  warnings: number;
  errors: number;
}

export interface CutoverPhase {
  phase: string;
  description: string;
  duration_days: number;
}

export interface OAuthClientMigrationConfig {
  source_system: "Auth0" | "Okta" | "Keycloak" | "Ping";
  migration_scope: { clients: boolean; users: boolean; policies: boolean; custom_claims: boolean };
  mapping_preview: MappingPreview[];
  validation_report: ValidationReport;
  phased_cutover_timeline: CutoverPhase[];
}

export function useOAuthClientMigrationConfig(baseUrl: string = "") {
  const [config, setConfig] = useState<OAuthClientMigrationConfig | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchConfig = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/oauth-client-migration-config`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setConfig(await res.json());
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); }
    finally { setLoading(false); }
  }, [baseUrl]);

  const updateConfig = useCallback(async (patch: Partial<OAuthClientMigrationConfig>) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/oauth-client-migration-config`, {
        method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(patch),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json(); setConfig(data); return data;
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); return null; }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { config, loading, error, fetchConfig, updateConfig };
}
