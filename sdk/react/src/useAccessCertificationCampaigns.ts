import { useState, useCallback, useEffect } from "react";

/**
 * DEMO DATA — Tries real API first, falls back to empty demo data.
 * isDemoData flag indicates whether live or fallback data is shown.
 */

export interface Campaign {
  id: string;
  name: string;
  scope: string;
  period: string;
  status: string;
  completion_pct: number;
}

export interface ReviewerWorkload {
  reviewer: string;
  assigned: number;
  completed: number;
  pending: number;
}

export interface PendingReview {
  id: string;
  user: string;
  role: string;
  last_accessed: string;
  reviewer: string;
}

export interface AccessCertificationCampaignsData {
  campaigns: Campaign[];
  reviewer_workload: ReviewerWorkload[];
  pending_reviews: PendingReview[];
  auto_escalation: string;
}

export function useAccessCertificationCampaigns() {
  const [data, setData] = useState<AccessCertificationCampaignsData | null>(null);
  const [isDemoData, setIsDemoData] = useState(true);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true); setError(null);
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
        campaigns: [
          { id: "c1", name: "Q3 Access Review", scope: "All Engineering Users", period: "Q3 2026", status: "active", completion_pct: 65 },
          { id: "c2", name: "Admin Privilege Review", scope: "Users with admin roles", period: "Monthly July", status: "active", completion_pct: 40 },
          { id: "c3", name: "Contractor Access Review", scope: "All contractor accounts", period: "Q2 2026", status: "completed", completion_pct: 100 },
          { id: "c4", name: "Service Account Audit", scope: "Non-human identities", period: "Annual 2026", status: "scheduled", completion_pct: 0 },
        ],
        reviewer_workload: [
          { reviewer: "Alice Chen", assigned: 45, completed: 30, pending: 15 },
          { reviewer: "Bob Smith", assigned: 38, completed: 25, pending: 13 },
          { reviewer: "Diana Liu", assigned: 52, completed: 40, pending: 12 },
          { reviewer: "Evan Park", assigned: 30, completed: 28, pending: 2 },
        ],
        pending_reviews: [
          { id: "r1", user: "svc.legacy-api", role: "admin", last_accessed: "3d ago", reviewer: "Alice Chen" },
          { id: "r2", user: "temp.contractor", role: "developer", last_accessed: "30d ago", reviewer: "Bob Smith" },
          { id: "r3", user: "automation-bot", role: "writer", last_accessed: "never", reviewer: "Diana Liu" },
        ],
        auto_escalation: "Escalate to manager after 7 days, to security lead after 14 days",
      });
    } catch (e) { setError(e instanceof Error ? e.message : "Failed"); }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);
  return { data, loading, error, refresh: fetchData, isDemoData };
}
