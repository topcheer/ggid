import { useState, useCallback, useEffect } from "react";

/**
 * DEMO DATA — Tries real API first, falls back to empty demo data.
 * isDemoData flag indicates whether live or fallback data is shown.
 */

export interface LockoutEntry {
  username: string;
  attempts: number;
  unlock_at: string;
}

export interface EndpointConfig {
  endpoint: string;
  threshold: number;
  lockout_min: number;
}

export interface LockoutPolicyConfigData {
  max_failed_attempts: number;
  lockout_duration_minutes: number;
  progressive_backoff: boolean;
  captcha_after: number;
  auto_unlock_after: number;
  current_lockouts: LockoutEntry[];
  per_endpoint: EndpointConfig[];
}

export function useLockoutPolicyConfig() {
  const [data, setData] = useState<LockoutPolicyConfigData | null>(null);
  const [isDemoData, setIsDemoData] = useState(true);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const fetchData = useCallback(async () => {
    setLoading(true); setError(null);
    try { // Try real API first
      let res: Response | null = null;
      try {
        res = await fetch("/api/v1/data", { headers: { "Content-Type": "application/json" } });
      } catch { res = null; }
      if (res?.ok) { const d = await res.json(); setData(d); setIsDemoData(false); return; }
      setIsDemoData(true);
      setData({ max_failed_attempts: 5, lockout_duration_minutes: 15, progressive_backoff: true, captcha_after: 3, auto_unlock_after: 30,
        current_lockouts: [{ username: "user_js", attempts: 5, unlock_at: "12m" }],
        per_endpoint: [{ endpoint: "/auth/login", threshold: 5, lockout_min: 15 }, { endpoint: "/auth/refresh", threshold: 10, lockout_min: 5 }, { endpoint: "/admin/*", threshold: 3, lockout_min: 60 }], });
    } catch (e) { setError(e instanceof Error ? e.message : "Failed"); } finally { setLoading(false); }
  }, []);
  useEffect(() => { fetchData(); }, [fetchData]);
  return { data, loading, error, refresh: fetchData, isDemoData };
}
