import { useState, useCallback } from "react";

export interface Assignment {
  id: string;
  reviewer_id: string;
  reviewer_name: string;
  assigned_users: number;
  strategy: "org_manager" | "role_based" | "round_robin";
  last_assigned: string;
}

export function useAutoAssignment(baseUrl: string = "") {
  const [assignments, setAssignments] = useState<Assignment[]>([]);
  const [campaigns, setCampaigns] = useState<string[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchCampaigns = useCallback(async () => {
    try {
      const res = await fetch(`${baseUrl}/api/v1/policy/auto-assignment/campaigns`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json(); setCampaigns(data.campaigns || data || []);
    } catch (e: any) { setError(e.message); }
  }, [baseUrl]);

  const fetchAssignments = useCallback(async (campaign: string) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/policy/auto-assignment?campaign=${encodeURIComponent(campaign)}`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json(); setAssignments(data.assignments || data || []);
    } catch (e: any) { setError(e.message); }
    finally { setLoading(false); }
  }, [baseUrl]);

  const reassign = useCallback(async (id: string) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/policy/auto-assignment/${id}/reassign`, { method: "POST" });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      return true;
    } catch (e: any) { setError(e.message); return false; }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { assignments, campaigns, loading, error, fetchCampaigns, fetchAssignments, reassign };
}
