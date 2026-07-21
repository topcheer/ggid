import { useState, useCallback } from "react";
export interface QueryResult { timestamp: string; user: string; action: string; resource: string; result: string; }
export interface AuditQuery { select: string[]; where: { field: string; operator: string; value: string }[]; group_by: string; order_by: string; limit: number; }
export function useQueryBuilder(baseUrl: string = "") {
  const [results, setResults] = useState<QueryResult[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const execute = useCallback(async (query: AuditQuery) => { setLoading(true); setError(null); try { const res = await fetch(baseUrl + "/api/v1/audit/query", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(query) }); if (!res.ok) throw new Error("HTTP " + res.status); const data = await res.json(); setResults(data.results || data || []); } catch (e: any) { setError(e.message); } finally { setLoading(false); } }, [baseUrl]);
  const saveQuery = useCallback(async (name: string, query: AuditQuery) => { setLoading(true); setError(null); try { const res = await fetch(baseUrl + "/api/v1/audit/query/save", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ name, query }) }); if (!res.ok) throw new Error("HTTP " + res.status); return true; } catch (e: any) { setError(e.message); return false; } finally { setLoading(false); } }, [baseUrl]);
  return { results, loading, error, execute, saveQuery };
}
