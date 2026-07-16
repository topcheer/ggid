import { useState, useCallback, useEffect } from "react";

/**
 * DEMO DATA — Tries real API first, falls back to empty demo data.
 * isDemoData flag indicates whether live or fallback data is shown.
 */

export interface ScheduledAudit {
  id: string;
  framework: string;
  frequency_cron: string;
  next_run: string;
  scope: string;
  owner: string;
}

export interface ChecklistItem {
  task: string;
  status: string;
}

export interface DeadlineItem {
  framework: string;
  description: string;
  days_left: number;
}

export interface OverdueAlert {
  framework: string;
  days_overdue: number;
}

export interface AuditComplianceSchedulerData {
  scheduled_audits: ScheduledAudit[];
  audit_preparation_checklist: ChecklistItem[];
  evidence_collection_status: { ready: number; total: number };
  upcoming_deadlines_30d: DeadlineItem[];
  overdue_alerts: OverdueAlert[];
}

export function useAuditComplianceScheduler() {
  const [data, setData] = useState<AuditComplianceSchedulerData | null>(null);
  const [isDemoData, setIsDemoData] = useState(true);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      // Try real API first
      let res: Response | null = null;
      try {
        res = await fetch("/api/v1/data", {
          headers: { "Content-Type": "application/json" },
        });
      } catch { res = null; }
      
      if (res?.ok) {
        const realData = await res.json();
        setData(realData);
        setIsDemoData(false);
        return;
      }
      
      // Fallback: empty demo data (no dangerous flags)
      setIsDemoData(true);
      setData({
        scheduled_audits: [
          { id: "sched-1", framework: "SOC2 Type II", frequency_cron: "0 0 1 */3 *", next_run: "2026-04-01", scope: "All production systems", owner: "Sarah Kim" },
          { id: "sched-2", framework: "ISO 27001", frequency_cron: "0 0 1 1 *", next_run: "2026-12-01", scope: "ISMS scope", owner: "Mike Lee" },
          { id: "sched-3", framework: "GDPR", frequency_cron: "0 0 1 */6 *", next_run: "2026-07-01", scope: "EU data processing", owner: "Anna Schmidt" },
          { id: "sched-4", framework: "HIPAA", frequency_cron: "0 0 1 * *", next_run: "2026-02-01", scope: "PHI handling", owner: "Dr. James Wong" },
        ],
        audit_preparation_checklist: [
          { task: "Collect access logs (90d)", status: "ready" },
          { task: "Generate policy compliance report", status: "ready" },
          { task: "Verify evidence chain of custody", status: "in_progress" },
          { task: "Review incident response records", status: "pending" },
          { task: "Export security training completion", status: "ready" },
        ],
        evidence_collection_status: { ready: 3, total: 5 },
        upcoming_deadlines_30d: [
          { framework: "HIPAA", description: "Annual compliance audit", days_left: 18 },
          { framework: "SOC2", description: "Quarterly evidence submission", days_left: 25 },
        ],
        overdue_alerts: [
          { framework: "PCI-DSS", days_overdue: 5 },
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
