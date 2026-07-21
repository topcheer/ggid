import { useState, useCallback } from "react";

export interface AffectedItem {
  id: string;
  type: "user" | "resource" | "role";
  name: string;
  current_access: string;
  projected_access: string;
  risk_level: "low" | "medium" | "high";
}

export interface PreviewResult {
  total_affected: number;
  gain_access: number;
  lose_access: number;
  no_change: boolean;
  items: AffectedItem[];
}

export function useImpactPreview(baseUrl: string = "") {
  const [result, setResult] = useState<PreviewResult | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const preview = useCallback(async (policyId: string, changes: string) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/policy/impact-preview`, { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ policy_id: policyId, changes }) });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setResult(await res.json());
    } catch (e: any) { setError(e.message); setResult(null); }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { result, loading, error, preview };
}
