import { useState, useCallback, useEffect } from "react";

export interface PipelineStage {
  stage: string;
  count: number;
  status: "active" | "idle";
}

export interface ReviewerAssignment {
  category: string;
  reviewers: string[];
}

export interface FreezeWindow {
  name: string;
  start: string;
  end: string;
  reason: string;
  active: boolean;
}

export interface EmergencyBypass {
  allowed: boolean;
  requires_approval: boolean;
  approvers: string[];
}

export interface SegregationOfDuties {
  enforced: boolean;
  description: string;
}

export interface AuditTrailEntry {
  policy_name: string;
  action: string;
  actor: string;
  timestamp: string;
}

export interface PolicyGovernanceWorkflowData {
  policy_change_pipeline: PipelineStage[];
  reviewer_assignment: ReviewerAssignment[];
  change_freeze_windows: FreezeWindow[];
  emergency_bypass: EmergencyBypass;
  segregation_of_duties: SegregationOfDuties;
  governance_audit_trail: AuditTrailEntry[];
}

export function usePolicyGovernanceWorkflow() {
  const [data, setData] = useState<PolicyGovernanceWorkflowData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      await new Promise((r) => setTimeout(r, 400));
      setData({
        policy_change_pipeline: [
          { stage: "draft", count: 5, status: "idle" },
          { stage: "review", count: 3, status: "active" },
          { stage: "approve", count: 2, status: "idle" },
          { stage: "activate", count: 0, status: "idle" },
        ],
        reviewer_assignment: [
          { category: "security", reviewers: ["security_team", "ciso"] },
          { category: "compliance", reviewers: ["compliance_team", "legal"] },
          { category: "access", reviewers: ["platform_admin", "security_team"] },
          { category: "infrastructure", reviewers: ["devops_lead"] },
        ],
        change_freeze_windows: [
          { name: "Holiday Freeze", start: "2026-12-20", end: "2027-01-05", reason: "Holiday change freeze", active: false },
          { name: "Maintenance Window", start: "2026-01-15 02:00", end: "2026-01-15 06:00", reason: "Quarterly maintenance", active: true },
        ],
        emergency_bypass: {
          allowed: true,
          requires_approval: true,
          approvers: ["ciso", "cto"],
        },
        segregation_of_duties: {
          enforced: true,
          description: "The person who drafts a policy change cannot be the same person who approves it.",
        },
        governance_audit_trail: [
          { policy_name: "MFA for Admin API", action: "approved", actor: "security_team", timestamp: "2h ago" },
          { policy_name: "Session Timeout Policy", action: "submitted", actor: "platform_admin", timestamp: "5h ago" },
          { policy_name: "IP Allowlist Finance", action: "rejected", actor: "ciso", timestamp: "1d ago" },
          { policy_name: "OAuth Scope Restriction", action: "drafted", actor: "dev_team", timestamp: "2d ago" },
          { policy_name: "Password Policy Update", action: "approved", actor: "compliance_team", timestamp: "3d ago" },
        ],
      });
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load data");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  return { data, loading, error, refresh: fetchData };
}
