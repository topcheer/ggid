import { useState, useCallback } from "react";

export interface ConsentEntry {
  key: string;
  label: string;
  granted: boolean;
  granted_at: string | null;
  version: number;
}

export interface ConsentData {
  user_id: string;
  username: string;
  consents: ConsentEntry[];
  history: { version: number; changed_at: string; changed_by: string; changes: string }[];
}

export function useConsentRegistry(baseUrl: string = "") {
  const [data, setData] = useState<ConsentData | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchConsent = useCallback(async (user: string) => {
    if (!user) return;
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/identity/consent?user=${encodeURIComponent(user)}`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setData(await res.json());
    } catch (e: any) { setError(e.message); setData(null); }
    finally { setLoading(false); }
  }, [baseUrl]);

  const updateConsent = useCallback(async (userId: string, consents: ConsentEntry[]) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/identity/consent/${userId}`, { method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ consents }) });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      return true;
    } catch (e: any) { setError(e.message); return false; }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { data, loading, error, fetchConsent, updateConsent };
}
