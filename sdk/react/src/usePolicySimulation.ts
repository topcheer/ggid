import { useState, useCallback } from "react";

export interface SimRule {
  id: string;
  effect: "allow" | "deny";
  resource: string;
  action: string;
  condition: string;
}

export interface SimResult {
  subject: string;
  resource: string;
  action: string;
  before: "allow" | "deny";
  after: "allow" | "deny";
  status: "would_allow" | "would_deny" | "unchanged";
}

export function usePolicySimulation(baseUrl: string = "") {
  const [results, setResults] = useState<SimResult[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const simulate = useCallback(async (rules: SimRule[]) => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/policy/simulate`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ rules }),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json();
      setResults(data.results || data || []);
      return true;
    } catch (e: any) {
      setError(e.message);
      return false;
    } finally {
      setLoading(false);
    }
  }, [baseUrl]);

  return { results, loading, error, simulate };
}
