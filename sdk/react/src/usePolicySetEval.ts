import { useState, useCallback } from "react";

export interface PolicyResult {
  policy_id: string;
  policy_name: string;
  decision: "allow" | "deny" | "no_match";
  matched_rule: string;
}

export interface EvalResponse {
  results: PolicyResult[];
  final_decision: string;
}

export function usePolicySetEval(baseUrl: string = "") {
  const [data, setData] = useState<EvalResponse | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const evaluate = useCallback(async (subject: string, resource: string, action: string = "access") => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/policy/set-evaluate`, { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ subject, resource, action }) });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setData(await res.json());
    } catch (e: any) { setError(e.message); setData(null); }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { data, loading, error, evaluate };
}
