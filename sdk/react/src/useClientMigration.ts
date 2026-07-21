import { useState, useCallback } from "react";

export interface ClientConfig {
  client_id: string;
  client_name: string;
  redirect_uris: string[];
  scopes: string[];
  grant_types: string[];
}

export interface DiffResult {
  redirect_uris: { added: string[]; removed: string[] };
  scopes: { added: string[]; removed: string[] };
  grant_types: { added: string[]; removed: string[] };
}

export function useClientMigration(baseUrl: string = "") {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const previewDiff = useCallback(async (original: ClientConfig, proposed: ClientConfig) => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/oauth/client-migration/preview`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ client_id: original.client_id, original, proposed }),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data: DiffResult = await res.json();
      return data;
    } catch (e: any) {
      setError(e.message);
      // local fallback
      return {
        redirect_uris: {
          added: proposed.redirect_uris.filter((x) => !original.redirect_uris.includes(x)),
          removed: original.redirect_uris.filter((x) => !proposed.redirect_uris.includes(x)),
        },
        scopes: {
          added: proposed.scopes.filter((x) => !original.scopes.includes(x)),
          removed: original.scopes.filter((x) => !proposed.scopes.includes(x)),
        },
        grant_types: {
          added: proposed.grant_types.filter((x) => !original.grant_types.includes(x)),
          removed: original.grant_types.filter((x) => !proposed.grant_types.includes(x)),
        },
      } as DiffResult;
    } finally {
      setLoading(false);
    }
  }, [baseUrl]);

  const executeMigration = useCallback(async (config: ClientConfig, gracePeriodDays: number = 7) => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/oauth/client-migration/execute`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ client_id: config.client_id, config, grace_period_days: gracePeriodDays }),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      return true;
    } catch (e: any) {
      setError(e.message);
      return false;
    } finally {
      setLoading(false);
    }
  }, [baseUrl]);

  return { loading, error, previewDiff, executeMigration };
}
