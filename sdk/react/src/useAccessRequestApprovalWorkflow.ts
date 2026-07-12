import { useState, useCallback, useEffect } from "react";

export interface ApprovalChainStep {
  role: string;
  status: "pending" | "approved" | "rejected" | "waiting";
}

export interface PendingRequest {
  id: string;
  requester_name: string;
  requested_role: string;
  sla_remaining_hours: number;
  auto_approve_eligible: boolean;
  approval_chain: ApprovalChainStep[];
}

export interface AutoApproveRule {
  id: string;
  name: string;
  condition: string;
  enabled: boolean;
}

export interface AccessRequestApprovalWorkflowData {
  pending_requests: PendingRequest[];
  auto_approve_rules: AutoApproveRule[];
}

export function useAccessRequestApprovalWorkflow() {
  const [data, setData] = useState<AccessRequestApprovalWorkflowData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      await new Promise((r) => setTimeout(r, 400));
      setData({
        pending_requests: [
          {
            id: "req-1",
            requester_name: "Alice Chen",
            requested_role: "Data Engineer",
            sla_remaining_hours: 6,
            auto_approve_eligible: false,
            approval_chain: [
              { role: "Manager", status: "approved" },
              { role: "Security Admin", status: "pending" },
              { role: "Compliance", status: "waiting" },
            ],
          },
          {
            id: "req-2",
            requester_name: "Bob Martinez",
            requested_role: "Read-Only Analyst",
            sla_remaining_hours: 2,
            auto_approve_eligible: true,
            approval_chain: [
              { role: "Manager", status: "pending" },
              { role: "Security Admin", status: "waiting" },
            ],
          },
          {
            id: "req-3",
            requester_name: "Carol Jones",
            requested_role: "Finance Admin",
            sla_remaining_hours: -1,
            auto_approve_eligible: false,
            approval_chain: [
              { role: "Manager", status: "approved" },
              { role: "Security Admin", status: "approved" },
              { role: "Compliance", status: "pending" },
            ],
          },
        ],
        auto_approve_rules: [
          { id: "rule-1", name: "Low-risk read roles", condition: "Role = Read-Only AND dept != Finance", enabled: true },
          { id: "rule-2", name: "Same-day contractor access", condition: "Contractor AND duration <= 8h", enabled: true },
          { id: "rule-3", name: "Dev environment access", condition: "Environment = dev AND role != admin", enabled: false },
        ],
      });
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load data");
    } finally {
      setLoading(false);
    }
  }, []);

  const approve = useCallback(async (reqId: string) => {
    console.log("Approving request:", reqId);
  }, []);

  const reject = useCallback(async (reqId: string) => {
    console.log("Rejecting request:", reqId);
  }, []);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  return { data, loading, error, refresh: fetchData, approve, reject };
}
