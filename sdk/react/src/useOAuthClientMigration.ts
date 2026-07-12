import { useState, useCallback } from "react";
export interface MigrationPreview { scopes_to_migrate: number; grants_to_migrate: number; tokens_to_migrate: number; conflicts: string[]; }
export function useOAuthClientMigration(baseUrl: string = "") {
  const [preview, setPreview] = useState<MigrationPreview | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const doPreview = useCallback(async (source: string, target: string) => { setLoading(true); setError(null); try { const res = await fetch(baseUrl + "/api/v1/oauth/client-migration/preview", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ source, target }) }); if (!res.ok) throw new Error("HTTP " + res.status); setPreview(await res.json()); } catch (e: any) { setError(e.message); } finally { setLoading(false); } }, [baseUrl]);
  return { preview, loading, error, doPreview };
}
