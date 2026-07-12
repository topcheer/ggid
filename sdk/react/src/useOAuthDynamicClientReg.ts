import { useState, useCallback } from "react";

export interface RegisteredClient {
  client_id: string;
  client_name: string;
  grant_types: string[];
  redirect_uris: string[];
  created_at: string;
}

export interface OAuthDynamicClientRegConfig {
  registration_endpoint: string;
  registered_clients: RegisteredClient[];
  software_statement_enabled: boolean;
}

export function useOAuthDynamicClientReg(baseUrl: string = "") {
  const [config, setConfig] = useState<OAuthDynamicClientRegConfig | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchConfig = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/oauth-dynamic-client-reg`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setConfig(await res.json());
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); }
    finally { setLoading(false); }
  }, [baseUrl]);

  const updateConfig = useCallback(async (patch: Partial<OAuthDynamicClientRegConfig>) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/oauth-dynamic-client-reg`, {
        method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(patch),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json(); setConfig(data); return data;
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); return null; }
    finally { setLoading(false); }
  }, [baseUrl]);

  const registerClient = useCallback(async (payload: Partial<RegisteredClient>) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/oauth-dynamic-client-reg/register`, {
        method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(payload),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json(); setConfig(data); return data;
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); return null; }
    finally { setLoading(false); }
  }, [baseUrl]);

  const deleteClient = useCallback(async (clientId: string) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/oauth-dynamic-client-reg/${clientId}`, { method: "DELETE" });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json(); setConfig(data); return data;
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); return null; }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { config, loading, error, fetchConfig, updateConfig, registerClient, deleteClient };
}
