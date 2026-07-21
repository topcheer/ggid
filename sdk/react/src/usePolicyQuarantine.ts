import { useState, useCallback } from "react";

export interface QuarantineResult {
  policy_id: string;
  policy_name: string;
  reason: string;
  duration_hours: number;
  affected_entities: { type: string; id: string; name: string }[];
  rollback_plan: { step: string; reversible: boolean }[];
  auto_reenable_at: string;
  quarantined: boolean;
}

export function usePolicyQuarantine(baseUrl: string = "") {
  const [data, setData] = useState<QuarantineResult | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const quarantine = useCallback(async (policyId: string, reason: string, durationHours: number) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/policy/quarantine`, { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ policy_id: policyId, reason, duration_hours: durationHours }) });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setData(await res.json()); return true;
    } catch (e: any) { setError(e.message); return false; }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { data, loading, error, quarantine };
}
