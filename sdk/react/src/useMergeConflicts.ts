import { useState, useCallback } from "react";

export interface ConflictRule {
  id: string;
  rule: string;
  resource: string;
  version_a_effect: string;
  version_b_effect: string;
  conflict_type: "contradictory" | "overlapping" | "redundant";
}

export function useMergeConflicts(baseUrl: string = "") {
  const [conflicts, setConflicts] = useState<ConflictRule[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchConflicts = useCallback(async (policyA: string, policyB: string) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/policy/merge-conflicts?a=${encodeURIComponent(policyA)}&b=${encodeURIComponent(policyB)}`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json(); setConflicts(data.conflicts || data || []);
    } catch (e: any) { setError(e.message); }
    finally { setLoading(false); }
  }, [baseUrl]);

  const resolve = useCallback(async (strategy: string) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/policy/merge-conflicts/resolve`, { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ strategy }) });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      return true;
    } catch (e: any) { setError(e.message); return false; }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { conflicts, loading, error, fetchConflicts, resolve };
}
