import { useState, useCallback } from "react";
export interface SimResult { subject: string; resource: string; action: string; current_decision: string; proposed_decision: string; changed: boolean; }
export function usePolicySimulationLab(baseUrl: string = "") {
  const [results, setResults] = useState<SimResult[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const simulate = useCallback(async (subject: string, resource: string, action: string) => {
    setLoading(true); setError(null);
    try { const res = await fetch(baseUrl + "/api/v1/policy/simulation-lab", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ subject, resource, action }) }); if (!res.ok) throw new Error("HTTP " + res.status); const data = await res.json(); setResults(data.results || []); return data; } catch (e: any) { setError(e.message); return null; } finally { setLoading(false); }
  }, [baseUrl]);
  return { results, loading, error, simulate };
}
