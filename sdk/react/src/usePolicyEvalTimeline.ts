import { useState, useCallback, useEffect } from "react";

/**
 * DEMO DATA — Tries real API first, falls back to empty demo data.
 */

export interface EvalStep {
  name: string;
  description: string;
  latency_ms: number;
}

export interface MatchedRule {
  rule_id: string;
  condition: string;
  matched: boolean;
}

export interface PolicyEvaluation {
  policy: string;
  total_eval_time_ms: number;
  cache_hit: boolean;
  decision: string;
  steps: EvalStep[];
  matched_rules: MatchedRule[];
}

export interface PolicyEvalTimelineData {
  evaluations: PolicyEvaluation[];
}

export function usePolicyEvalTimeline() {
  const [data, setData] = useState<PolicyEvalTimelineData | null>(null);
  const [isDemoData, setIsDemoData] = useState(true);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      // Try real API first
      let res: Response | null = null;
      try { res = await fetch("/api/v1/data", { headers: { "Content-Type": "application/json" } }); } catch { res = null; }
      if (res?.ok) { const d = await res.json(); setData(d); setIsDemoData(false); return; }
      setIsDemoData(true);
      setData({
        evaluations: [
          { policy: "admin-access-policy", total_eval_time_ms: 42, cache_hit: true, decision: "allow", steps: [
            { name: "Request Received", description: "HTTP request parsed, subject extracted", latency_ms: 2 },
            { name: "Context Resolve", description: "User/resource/env attributes fetched", latency_ms: 8 },
            { name: "Rule Match", description: "3 rules evaluated against context", latency_ms: 15 },
            { name: "Decision", description: "Policy decision: allow", latency_ms: 5 },
            { name: "Response", description: "Decision serialized and returned", latency_ms: 12 },
          ], matched_rules: [
            { rule_id: "R001", condition: "subject.role == admin AND resource.type == dashboard", matched: true },
            { rule_id: "R002", condition: "time.in_business_hours()", matched: true },
            { rule_id: "R003", condition: "device.trust_level >= medium", matched: false },
          ] },
          { policy: "api-access-policy", total_eval_time_ms: 128, cache_hit: false, decision: "deny", steps: [
            { name: "Request Received", description: "HTTP request parsed", latency_ms: 3 },
            { name: "Context Resolve", description: "Attributes fetched from DB (cache miss)", latency_ms: 65 },
            { name: "Rule Match", description: "5 rules evaluated", latency_ms: 35 },
            { name: "Decision", description: "Policy decision: deny", latency_ms: 5 },
            { name: "Response", description: "403 returned", latency_ms: 20 },
          ], matched_rules: [
            { rule_id: "R010", condition: "scope includes read:api", matched: false },
          ] },
        ],
      });
    } catch (e) { setError(e instanceof Error ? e.message : "Failed"); }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);
  return { data, loading, error, refresh: fetchData, isDemoData };
}
