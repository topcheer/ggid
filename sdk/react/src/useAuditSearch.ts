import { useState, useCallback } from "react";

export interface AuditEntry {
  id: string;
  timestamp: string;
  user: string;
  action: string;
  resource: string;
  severity: "info" | "warning" | "error" | "critical";
  detail: string;
}

export function useAuditSearch(baseUrl: string = "") {
  const [results, setResults] = useState<AuditEntry[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [page, setPage] = useState(1);
  const [totalPages, setTotalPages] = useState(1);

  const search = useCallback(async (query: string, filters: Record<string, string>, p: number = 1) => {
    setLoading(true); setError(null);
    try {
      const params = new URLSearchParams({ q: query, page: String(p), ...Object.fromEntries(Object.entries(filters).filter(([, v]) => v)) });
      const res = await fetch(baseUrl + "/api/v1/audit/search?" + params);
      if (!res.ok) throw new Error("HTTP " + res.status);
      const data = await res.json(); setResults(data.entries || data.results || data || []); setTotalPages(data.total_pages || 1); setPage(p);
    } catch (e: any) { setError(e.message); }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { results, loading, error, page, totalPages, search };
}
