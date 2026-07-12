import { useState, useCallback } from "react";

export interface BackchannelLogoutClient {
  client_id: string;
  client_name: string;
  logout_endpoint_url: string;
}

export interface LogoutErrorHandling {
  retry_attempts: number;
  timeout_seconds: number;
  failed_24h: number;
}

export interface OidcBackchannelLogoutConfig {
  per_client_endpoints: BackchannelLogoutClient[];
  session_lifetime_after_logout: number;
  token_revocation_on_logout: boolean;
  logout_token_preview: string;
  error_handling: LogoutErrorHandling;
}

export function useOidcBackchannelLogoutConfig(baseUrl: string = "") {
  const [config, setConfig] = useState<OidcBackchannelLogoutConfig | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchConfig = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/oidc-backchannel-logout-config`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setConfig(await res.json());
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); }
    finally { setLoading(false); }
  }, [baseUrl]);

  const updateConfig = useCallback(async (patch: Partial<OidcBackchannelLogoutConfig>) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/oidc-backchannel-logout-config`, {
        method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(patch),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json(); setConfig(data); return data;
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); return null; }
    finally { setLoading(false); }
  }, [baseUrl]);

  const testLogout = useCallback(async (clientId: string) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/oidc-backchannel-logout-config/test`, {
        method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ client_id: clientId }),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      return await res.json();
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); return null; }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { config, loading, error, fetchConfig, updateConfig, testLogout };
}
