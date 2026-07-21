import { useState, useCallback } from "react";

export interface ConflictPair {
  id: string;
  policy_a: string;
  policy_b: string;
  overlap_type: "resource" | "action" | "subject" | "rule";
  severity: "low" | "medium" | "high" | "critical";
  resource_pattern: string;
  description: string;
}

export function usePolicyConflicts(baseUrl: string = "") {
  const [conflicts, setConflicts] = useState<ConflictPair[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchConflicts = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/policy/conflicts`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json(); setConflicts(data.conflicts || data || []);
    } catch (e: any) { setError(e.message); }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { conflicts, loading, error, fetchConflicts };
}
