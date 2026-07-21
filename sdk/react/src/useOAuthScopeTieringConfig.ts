import { useState, useCallback } from "react";

export interface TierDefinition {
  tier: string;
  consent_policy: "none" | "implicit" | "explicit" | "admin_required";
}

export interface ScopePackage {
  name: string;
  scopes: string[];
}

export interface ScopeInheritanceRule {
  parent_scope: string;
  child_scopes: string[];
}

export interface OAuthScopeTieringConfig {
  tier_definitions: TierDefinition[];
  scope_packages: ScopePackage[];
  scope_inheritance_rules: ScopeInheritanceRule[];
  least_privilege_defaults: boolean;
  migration_from_flat_scopes: boolean;
}

export function useOAuthScopeTieringConfig(baseUrl: string = "") {
  const [config, setConfig] = useState<OAuthScopeTieringConfig | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const fetchConfig = useCallback(async () => {
    setLoading(true); setError(null);
    try { const res = await fetch(`${baseUrl}/api/v1/settings/oauth-scope-tiering-config`); if (!res.ok) throw new Error(`HTTP ${res.status}`); setConfig(await res.json()); }
    catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); } finally { setLoading(false); }
  }, [baseUrl]);
  const updateConfig = useCallback(async (patch: Partial<OAuthScopeTieringConfig>) => {
    setLoading(true); setError(null);
    try { const res = await fetch(`${baseUrl}/api/v1/settings/oauth-scope-tiering-config`, { method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(patch) }); if (!res.ok) throw new Error(`HTTP ${res.status}`); const data = await res.json(); setConfig(data); return data; }
    catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); return null; } finally { setLoading(false); }
  }, [baseUrl]);
  return { config, loading, error, fetchConfig, updateConfig };
}
