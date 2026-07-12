import { useState, useCallback } from "react";

export interface ScopeConsentEntry {
  scope: string;
  consent_level: "none" | "implicit" | "explicit" | "admin_required";
  risk_level: "low" | "medium" | "high";
}

export interface OAuthScopeConsentMatrixConfig {
  matrix: ScopeConsentEntry[];
  compliance_summary_pct: number;
}

export function useOAuthScopeConsentMatrixConfig(baseUrl: string = "") {
  const [config, setConfig] = useState<OAuthScopeConsentMatrixConfig | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchConfig = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/oauth-scope-consent-matrix-config`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setConfig(await res.json());
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); }
    finally { setLoading(false); }
  }, [baseUrl]);

  const updateConfig = useCallback(async (patch: Partial<OAuthScopeConsentMatrixConfig>) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/oauth-scope-consent-matrix-config`, {
        method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(patch),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json(); setConfig(data); return data;
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); return null; }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { config, loading, error, fetchConfig, updateConfig };
}
