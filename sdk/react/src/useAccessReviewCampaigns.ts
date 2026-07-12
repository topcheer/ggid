import { useState, useCallback } from "react";

export interface Campaign {
  id: string;
  name: string;
  scope: string;
  reviewers: string[];
  deadline: string;
  completion_pct: number;
  auto_revoke: boolean;
  reminders: boolean;
  status: "active" | "completed" | "overdue";
}

export function useAccessReviewCampaigns(baseUrl: string = "") {
  const [campaigns, setCampaigns] = useState<Campaign[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchCampaigns = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/policy/access-review-campaigns`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json(); setCampaigns(data.campaigns || data || []);
    } catch (e: any) { setError(e.message); }
    finally { setLoading(false); }
  }, [baseUrl]);

  const createCampaign = useCallback(async (payload: { name: string; scope: string; reviewers: string[]; deadline: string; auto_revoke: boolean }) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/policy/access-review-campaigns`, { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(payload) });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      return true;
    } catch (e: any) { setError(e.message); return false; }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { campaigns, loading, error, fetchCampaigns, createCampaign };
}
