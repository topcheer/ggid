import { useState, useCallback } from "react";
export interface ExplainResult { decision: string; confidence: number; matched_rules: { rule: string; effect: string; priority: number }[]; contributing_factors: string[]; alternatives: { policy: string; decision: string }[]; eval_path: string[]; }
export function usePolicyDecisionExplain(baseUrl: string = "") {
  const [result, setResult] = useState<ExplainResult | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const explain = useCallback(async (subject: string, resource: string, action: string) => { setLoading(true); setError(null); try { const res = await fetch(baseUrl + "/api/v1/policy/decision-explain", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ subject, resource, action }) }); if (!res.ok) throw new Error("HTTP " + res.status); setResult(await res.json()); } catch (e: any) { setError(e.message); } finally { setLoading(false); } }, [baseUrl]);
  return { result, loading, error, explain };
}
