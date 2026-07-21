import { useState, useCallback } from "react";

export interface DryRunResult {
  decision: "allow" | "deny" | "no_match";
  matched_rules: { rule_id: string; rule_name: string; effect: string }[];
  explanation: string;
  decision_time_ms: number;
}

export function usePolicyDryRun(baseUrl: string = "") {
  const [result, setResult] = useState<DryRunResult | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const evaluate = useCallback(async (policyId: string, subject: string, resource: string, action: string = "access") => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(baseUrl + "/api/v1/policy/dry-run", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ policy_id: policyId, subject, resource, action }) });
      if (!res.ok) throw new Error("HTTP " + res.status);
      setResult(await res.json());
    } catch (e: any) { setError(e.message); setResult(null); }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { result, loading, error, evaluate };
}
