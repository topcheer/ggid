import { useState, useCallback } from "react";

export interface BreachResult {
  id: string;
  username: string;
  breach_count: number;
  breach_sources: string[];
  last_checked: string;
  severity: "clean" | "low" | "medium" | "high" | "critical";
}

export function usePasswordBreachCheck(baseUrl: string = "") {
  const [results, setResults] = useState<BreachResult[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchResults = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(baseUrl + "/api/v1/auth/password-breach-check");
      if (!res.ok) throw new Error("HTTP " + res.status);
      const data = await res.json(); setResults(data.results || data || []);
    } catch (e: any) { setError(e.message); }
    finally { setLoading(false); }
  }, [baseUrl]);

  const scanNow = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(baseUrl + "/api/v1/auth/password-breach-check/scan", { method: "POST" });
      if (!res.ok) throw new Error("HTTP " + res.status);
      return true;
    } catch (e: any) { setError(e.message); return false; }
    finally { setLoading(false); }
  }, [baseUrl]);

  const forceReset = useCallback(async (id: string) => {
    try { await fetch(baseUrl + "/api/v1/auth/password-breach-check/" + id + "/force-reset", { method: "POST" }); return true; }
    catch { return false; }
  }, [baseUrl]);

  return { results, loading, error, fetchResults, scanNow, forceReset };
}
