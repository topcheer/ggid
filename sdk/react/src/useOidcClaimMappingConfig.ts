import { useState, useCallback, useEffect } from "react";

export interface ClaimEntry {
  name: string;
  source: string;
  transform: string;
  token_type: string;
}

export interface ScopeRow {
  scope: string;
  claims: string[];
}

export interface OidcClaimMappingConfigData {
  claims: ClaimEntry[];
  all_claims: string[];
  scope_matrix: ScopeRow[];
}

export function useOidcClaimMappingConfig() {
  const [data, setData] = useState<OidcClaimMappingConfigData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const fetchData = useCallback(async () => {
    setLoading(true); setError(null);
    try { await new Promise((r) => setTimeout(r, 400));
      setData({ claims: [
        { name: "sub", source: "user_attr:id", transform: "direct", token_type: "id_token+access" },
        { name: "email", source: "user_attr:email", transform: "direct", token_type: "id_token" },
        { name: "groups", source: "group", transform: "flatten", token_type: "access_token" },
        { name: "roles", source: "group:role", transform: "regex:^role_(.+)", token_type: "access_token" },
        { name: "tenant", source: "static", transform: "constant:default", token_type: "both" },
      ], all_claims: ["sub", "email", "groups", "roles", "tenant"],
        scope_matrix: [
          { scope: "openid", claims: ["sub"] },
          { scope: "profile", claims: ["sub", "groups"] },
          { scope: "email", claims: ["sub", "email"] },
          { scope: "admin", claims: ["sub", "groups", "roles", "tenant"] },
        ] });
    } catch (e) { setError(e instanceof Error ? e.message : "Failed"); } finally { setLoading(false); }
  }, []);
  useEffect(() => { fetchData(); }, [fetchData]);
  return { data, loading, error, refresh: fetchData };
}
