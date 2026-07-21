import { useState, useCallback, useEffect } from "react";

/**
 * DEMO DATA — Tries real API first, falls back to empty demo data.
 */

export interface PolicyByStatus {
  draft: number;
  active: number;
  quarantined: number;
  deprecated: number;
}

export interface ApprovalPipeline {
  submitted: number;
  reviewing: number;
  approved: number;
  active: number;
}

export interface PolicyChange {
  policy_name: string;
  action: string;
  author: string;
  timestamp: string;
}

export interface PolicyAgeBin {
  range: string;
  count: number;
}

export interface PolicyLifecycleDashboardData {
  policies_by_status: PolicyByStatus;
  approval_pipeline: ApprovalPipeline;
  avg_approval_time_hours: number;
  recent_changes: PolicyChange[];
  rollback_count: number;
  policy_age_histogram: PolicyAgeBin[];
}

export function usePolicyLifecycleDashboard() {
  const [data, setData] = useState<PolicyLifecycleDashboardData | null>(null);
  const [isDemoData, setIsDemoData] = useState(true);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      // Try real API first
      let res: Response | null = null;
      try { res = await fetch("/api/v1/data", { headers: { "Content-Type": "application/json" } }); } catch { res = null; }
      if (res?.ok) { const d = await res.json(); setData(d); setIsDemoData(false); return; }
      setIsDemoData(true);
      setData({
        policies_by_status: { draft: 8, active: 42, quarantined: 3, deprecated: 12 },
        approval_pipeline: { submitted: 5, reviewing: 3, approved: 2, active: 42 },
        avg_approval_time_hours: 36,
        recent_changes: [
          { policy_name: "MFA Required for Admin API", action: "activated", author: "security_team", timestamp: "2h ago" },
          { policy_name: "Session Timeout Policy", action: "updated", author: "platform_admin", timestamp: "5h ago" },
          { policy_name: "IP Allowlist for Finance", action: "quarantined", author: "compliance", timestamp: "1d ago" },
          { policy_name: "OAuth Scope Restriction", action: "drafted", author: "dev_team", timestamp: "2d ago" },
        ],
        rollback_count: 2,
        policy_age_histogram: [
          { range: "<7d", count: 6 },
          { range: "1-4w", count: 15 },
          { range: "1-3m", count: 22 },
          { range: "3-6m", count: 12 },
          { range: "6m+", count: 10 },
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

  return { data, loading, error, refresh: fetchData, isDemoData };
}
