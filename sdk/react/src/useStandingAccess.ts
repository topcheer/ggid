import { useState, useCallback } from "react";

export interface StandingAccessEntry {
  id: string;
  user_id: string;
  username: string;
  resource: string;
  access_type: string;
  granted_at: string;
  last_used: string;
  days_since_use: number;
  jit_recommended: boolean;
  jit_role: string;
}

export function useStandingAccess(baseUrl: string = "") {
  const [entries, setEntries] = useState<StandingAccessEntry[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchEntries = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/policy/standing-access`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json();
      setEntries(data.entries || data || []);
    } catch (e: any) {
      setError(e.message);
    } finally {
      setLoading(false);
    }
  }, [baseUrl]);

  const convertJIT = useCallback(async (id: string) => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/policy/standing-access/${id}/convert-jit`, {
        method: "POST",
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setEntries((prev: any) => prev.filter((e: any) => e.id !== id));
      return true;
    } catch (e: any) {
      setError(e.message);
      return false;
    } finally {
      setLoading(false);
    }
  }, [baseUrl]);

  return { entries, loading, error, fetchEntries, convertJIT };
}
