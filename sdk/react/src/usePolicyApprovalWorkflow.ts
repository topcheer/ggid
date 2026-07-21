import { useState, useCallback } from "react";

export interface PolicyApproval {
  id: string;
  policy_name: string;
  requested_by: string;
  risk_level: "low" | "medium" | "high" | "critical";
  submitted_at: string;
  expires_at: string;
  days_remaining: number;
  change_summary: string;
  approval_chain: { approver: string; status: string; acted_at: string | null }[];
  comments: { author: string; text: string; timestamp: string }[];
}

export function usePolicyApprovalWorkflow(baseUrl: string = "") {
  const [approvals, setApprovals] = useState<PolicyApproval[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchApprovals = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(baseUrl + "/api/v1/policy/approval-workflow");
      if (!res.ok) throw new Error("HTTP " + res.status);
      const data = await res.json(); setApprovals(data.approvals || data || []);
    } catch (e: any) { setError(e.message); }
    finally { setLoading(false); }
  }, [baseUrl]);

  const decide = useCallback(async (id: string, decision: string, comment: string) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(baseUrl + "/api/v1/policy/approval-workflow/" + id, { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ decision, comment }) });
      if (!res.ok) throw new Error("HTTP " + res.status);
      return true;
    } catch (e: any) { setError(e.message); return false; }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { approvals, loading, error, fetchApprovals, decide };
}
