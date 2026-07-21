import { useState, useCallback } from "react";

export interface CertificationUser {
  user_id: string;
  username: string;
  email: string;
  current_role: string;
  last_login: string;
  status: "pending" | "certified" | "revoked" | "modified";
  comment: string;
}

export interface Campaign {
  id: string;
  name: string;
  framework: string;
  deadline: string;
  total_users: number;
  completed: number;
}

export type CertificationDecision = "certified" | "revoked" | "modified";

export function useAccessCertification(baseUrl: string = "") {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchCampaigns = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/policy/access-certification/campaigns`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json();
      return (data.campaigns || data || []) as Campaign[];
    } catch (e: any) {
      setError(e.message);
      return [];
    } finally {
      setLoading(false);
    }
  }, [baseUrl]);

  const fetchUsers = useCallback(async (campaignId: string) => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/policy/access-certification/campaigns/${campaignId}/users`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json();
      return (data.users || data || []) as CertificationUser[];
    } catch (e: any) {
      setError(e.message);
      return [];
    } finally {
      setLoading(false);
    }
  }, [baseUrl]);

  const submitDecision = useCallback(async (campaignId: string, userId: string, decision: CertificationDecision, comment?: string) => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/policy/access-certification/campaigns/${campaignId}/users/${userId}`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ decision, comment }),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      return true;
    } catch (e: any) {
      setError(e.message);
      return false;
    } finally {
      setLoading(false);
    }
  }, [baseUrl]);

  return { loading, error, fetchCampaigns, fetchUsers, submitDecision };
}
