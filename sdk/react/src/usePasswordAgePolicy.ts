import { useState, useCallback } from "react";

export interface PasswordPolicy {
  max_age_days: number;
  expiry_warning_days: number;
  enforce_after: boolean;
  per_org_override: { org_id: string; org_name: string; max_age_days: number; enabled: boolean }[];
  upcoming_expiry: { user_id: string; username: string; org: string; expires_in_days: number }[];
}

export function usePasswordAgePolicy(baseUrl: string = "") {
  const [data, setData] = useState<PasswordPolicy | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchPolicy = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/auth/password-age-policy`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setData(await res.json());
    } catch (e: any) { setError(e.message); }
    finally { setLoading(false); }
  }, [baseUrl]);

  const savePolicy = useCallback(async (policy: PasswordPolicy) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/auth/password-age-policy`, { method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(policy) });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setData(policy); return true;
    } catch (e: any) { setError(e.message); return false; }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { data, loading, error, fetchPolicy, savePolicy };
}
