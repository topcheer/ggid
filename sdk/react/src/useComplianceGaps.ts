import { useState, useEffect, useCallback } from "react";

export interface ComplianceGap {
  id: string;
  control_id: string;
  framework: string;
  description: string;
  remediation_plan: string;
  owner: string;
  due_date: string;
  status: "open" | "in_progress" | "remediated" | "accepted_risk";
  severity: "low" | "medium" | "high" | "critical";
}

export interface ComplianceGapsResult {
  gaps: ComplianceGap[];
}

export function useComplianceGaps(baseUrl: string = "") {
  const [gaps, setGaps] = useState<ComplianceGap[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchGaps = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/audit/compliance-gaps`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data: ComplianceGapsResult = await res.json();
      setGaps(data.gaps || data as any || []);
    } catch (e: any) {
      setError(e.message);
    } finally {
      setLoading(false);
    }
  }, [baseUrl]);

  const updateStatus = useCallback(async (gapId: string, status: string) => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/audit/compliance-gaps/${gapId}`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ status }),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setGaps((prev: any) => prev.map((g) => g.id === gapId ? { ...g, status: status as ComplianceGap["status"] } : g));
      return true;
    } catch (e: any) {
      setError(e.message);
      return false;
    } finally {
      setLoading(false);
    }
  }, [baseUrl]);

  useEffect(() => {
    fetchGaps();
  }, [fetchGaps]);

  return { gaps, loading, error, fetchGaps, updateStatus };
}
