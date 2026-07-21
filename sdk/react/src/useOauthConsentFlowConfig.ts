import { useState, useCallback } from "react";

export interface ScopeDescription {
  scope: string;
  description: string;
}

export interface ConsentScreenConfig {
  logo_url: string;
  privacy_url: string;
  tos_url: string;
}

export interface PreApprovedApp {
  client_id: string;
  client_name: string;
  scopes: string[];
}

export interface OauthConsentFlowConfig {
  consent_screen: ConsentScreenConfig;
  per_scope_description: ScopeDescription[];
  show_skip_consent: boolean;
  remember_duration_days: number;
  pre_approved_apps: PreApprovedApp[];
}

export function useOauthConsentFlowConfig(baseUrl: string = "") {
  const [config, setConfig] = useState<OauthConsentFlowConfig | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchConfig = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/oauth-consent-flow-config`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setConfig(await res.json());
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); }
    finally { setLoading(false); }
  }, [baseUrl]);

  const updateConfig = useCallback(async (patch: Partial<OauthConsentFlowConfig>) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/oauth-consent-flow-config`, {
        method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(patch),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json(); setConfig(data); return data;
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); return null; }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { config, loading, error, fetchConfig, updateConfig };
}
