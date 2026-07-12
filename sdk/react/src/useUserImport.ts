import { useState, useCallback } from "react";
export interface ImportResults { created: number; updated: number; skipped: number; failed: number; errors: string[]; }
export function useUserImport(baseUrl: string = "") {
  const [results, setResults] = useState<ImportResults | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const importUsers = useCallback(async (file: string, mappings: { csv_column: string; user_attribute: string }[], dryRun: boolean) => { setLoading(true); setError(null); try { const res = await fetch(baseUrl + "/api/v1/identity/users/import", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ file, mappings, dry_run: dryRun }) }); if (!res.ok) throw new Error("HTTP " + res.status); const data = await res.json(); setResults(data); return data; } catch (e: any) { setError(e.message); return null; } finally { setLoading(false); } }, [baseUrl]);
  return { results, loading, error, importUsers };
}
