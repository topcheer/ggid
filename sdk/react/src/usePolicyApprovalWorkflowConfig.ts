import { useState, useCallback, useEffect } from "react";

export interface PipelineStage {
  name: string;
  assignee: string;
  enabled: boolean;
}

export interface ReviewerAssignment {
  category: string;
  reviewer: string;
}

export interface FreezeWindow {
  name: string;
  period: string;
}

export interface PolicyApprovalWorkflowConfigData {
  pipeline: PipelineStage[];
  reviewers: ReviewerAssignment[];
  freeze_windows: FreezeWindow[];
  sod_enforced: boolean;
  emergency_bypass_enabled: boolean;
}

export function usePolicyApprovalWorkflowConfig() {
  const [data, setData] = useState<PolicyApprovalWorkflowConfigData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const fetchData = useCallback(async () => {
    setLoading(true); setError(null);
    try { await new Promise((r) => setTimeout(r, 400));
      setData({
        pipeline: [
          { name: "Draft", assignee: "Requester", enabled: true },
          { name: "Review", assignee: "Security Team", enabled: true },
          { name: "Approve", assignee: "Policy Owner", enabled: true },
          { name: "Activate", assignee: "System", enabled: true },
        ],
        reviewers: [
          { category: "access_control", reviewer: "Alice (Security)" },
          { category: "data_protection", reviewer: "Bob (DPO)" },
          { category: "compliance", reviewer: "Carol (Compliance)" },
        ],
        freeze_windows: [
          { name: "Year-end Freeze", period: "Dec 20 - Jan 5" },
          { name: "Quarterly Freeze", period: "Last 3 days of quarter" },
        ],
        sod_enforced: true, emergency_bypass_enabled: true,
      });
    } catch (e) { setError(e instanceof Error ? e.message : "Failed"); } finally { setLoading(false); }
  }, []);
  useEffect(() => { fetchData(); }, [fetchData]);
  return { data, loading, error, refresh: fetchData };
}
