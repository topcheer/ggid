import { useState, useCallback } from "react";

export interface TransferImpact {
  roles_revoked: string[];
  default_role_assigned: string;
  managers_notified: string[];
  policies_affected: number;
  sessions_revoked: number;
}

export function useOrgTransfer(baseUrl: string = "") {
  const [impact, setImpact] = useState<TransferImpact | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const preview = useCallback(async (userId: string, newOrgId: string) => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/identity/org-transfer/preview`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ user_id: userId, new_org_id: newOrgId }),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data: TransferImpact = await res.json();
      setImpact(data);
      return data;
    } catch (e: any) {
      setError(e.message);
      return null;
    } finally {
      setLoading(false);
    }
  }, [baseUrl]);

  const execute = useCallback(async (userId: string, newOrgId: string) => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/identity/org-transfer/execute`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ user_id: userId, new_org_id: newOrgId }),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setImpact(null);
      return true;
    } catch (e: any) {
      setError(e.message);
      return false;
    } finally {
      setLoading(false);
    }
  }, [baseUrl]);

  return { impact, loading, error, preview, execute };
}
