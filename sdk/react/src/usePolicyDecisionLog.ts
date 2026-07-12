import { useState, useCallback } from "react";

export interface PolicyDecision {
  id: string;
  timestamp: string;
  policy_id: string;
  subject: string;
  resource: string;
  action: string;
  decision: "allow" | "deny";
  matched_rules: string[];
  evaluation_time_ms: number;
}

export function usePolicyDecisionLog(baseUrl: string = "") {
  const [decisions, setDecisions] = useState<PolicyDecision[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchDecisions = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(baseUrl + "/api/v1/policy/decision-log");
      if (!res.ok) throw new Error("HTTP " + res.status);
      const data = await res.json(); setDecisions(data.decisions || data || []);
    } catch (e: any) { setError(e.message); }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { decisions, loading, error, fetchDecisions };
}
