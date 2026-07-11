import { useState, useCallback } from "react";

export interface ClientVersion {
  version: number;
  config_hash: string;
  redirect_uris: string[];
  scopes: string[];
  grant_types: string[];
  created_at: string;
  created_by: string;
  change_description: string;
}

export function useClientVersioning(baseUrl: string = "") {
  const [versions, setVersions] = useState<ClientVersion[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchVersions = useCallback(async (clientId: string) => {
    if (!clientId) return;
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/oauth/clients/${clientId}/versions`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json();
      setVersions(data.versions || data || []);
    } catch (e: any) {
      setError(e.message);
    } finally {
      setLoading(false);
    }
  }, [baseUrl]);

  const rollback = useCallback(async (clientId: string, version: number) => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/oauth/clients/${clientId}/rollback`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ version }),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      await fetchVersions(clientId);
      return true;
    } catch (e: any) {
      setError(e.message);
      return false;
    } finally {
      setLoading(false);
    }
  }, [baseUrl, fetchVersions]);

  return { versions, loading, error, fetchVersions, rollback };
}
