import { useState, useCallback } from "react";

export interface PrivilegedAccount {
  id: string;
  user_id: string;
  username: string;
  email: string;
  roles: string[];
  granted_at: string;
  justification: string;
  expires_at: string;
  days_until_expiry: number;
}

export function usePrivilegedAccess(baseUrl: string = "") {
  const [accounts, setAccounts] = useState<PrivilegedAccount[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchAccounts = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/policy/privileged-access`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json();
      setAccounts(data.accounts || data || []);
    } catch (e: any) {
      setError(e.message);
    } finally {
      setLoading(false);
    }
  }, [baseUrl]);

  const batchRevoke = useCallback(async (ids: string[]) => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/policy/privileged-access/batch-revoke`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ ids }),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setAccounts((prev: any) => prev.filter((a) => !ids.includes(a.id)));
      return true;
    } catch (e: any) {
      setError(e.message);
      return false;
    } finally {
      setLoading(false);
    }
  }, [baseUrl]);

  return { accounts, loading, error, fetchAccounts, batchRevoke };
}
