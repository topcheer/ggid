import { useState, useCallback, useEffect } from "react";

export interface RemediationItem {
  source: string;
  finding: string;
  severity: string;
  assignee: string;
  due_date: string;
  status: string;
  progress_pct: number;
}

export interface OverdueAlert {
  finding_id: string;
  source: string;
  days_overdue: number;
  assignee: string;
}

export interface TeamBreakdown {
  team: string;
  total: number;
  completed: number;
}

export interface RemediationTrackerData {
  remediation_items: RemediationItem[];
  overdue_alerts: OverdueAlert[];
  completion_rate_pct: number;
  per_team_breakdown: TeamBreakdown[];
}

export function useRemediationTracker() {
  const [data, setData] = useState<RemediationTrackerData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      await new Promise((r) => setTimeout(r, 400));
      setData({
        remediation_items: [
          { source: "vuln", finding: "CVE-2024-1234: RCE in openssl", severity: "critical", assignee: "infra-team", due_date: "2024-03-15", status: "in_progress", progress_pct: 60 },
          { source: "pentest", finding: "SSRF in webhook delivery", severity: "high", assignee: "backend-team", due_date: "2024-03-12", status: "overdue", progress_pct: 30 },
          { source: "audit", finding: "Missing access review documentation", severity: "medium", assignee: "sec-ops", due_date: "2024-03-20", status: "in_progress", progress_pct: 45 },
          { source: "compliance", finding: "PCI-DSS Req 8.3 MFA enforcement", severity: "high", assignee: "identity-team", due_date: "2024-03-08", status: "completed", progress_pct: 100 },
          { source: "vuln", finding: "CVE-2024-9012: Info disclosure", severity: "medium", assignee: "backend-team", due_date: "2024-04-01", status: "open", progress_pct: 0 },
        ],
        overdue_alerts: [
          { finding_id: "PENT-005", source: "pentest", days_overdue: 5, assignee: "backend-team" },
        ],
        completion_rate_pct: 20,
        per_team_breakdown: [
          { team: "infra", total: 1, completed: 0 },
          { team: "backend", total: 2, completed: 0 },
          { team: "sec-ops", total: 1, completed: 0 },
          { team: "identity", total: 1, completed: 1 },
        ],
      });
    } catch (e) { setError(e instanceof Error ? e.message : "Failed"); }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);
  return { data, loading, error, refresh: fetchData };
}
