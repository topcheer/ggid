import { useState, useCallback } from "react";

export interface ConsentInfo {
  client_id: string;
  client_name: string;
  user_count: number;
  scopes: string[];
  last_granted: string;
  consent_rate: number;
}

export interface ConsentDashboardData {
  active_consents: ConsentInfo[];
  revocation_trend: { day: string; count: number }[];
  pending_expiry: { client_name: string; user: string; expires_at: string; days_left: number }[];
}

export function useConsentDashboard(baseUrl: string = "") {
  const [data, setData] = useState<ConsentDashboardData | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(baseUrl + "/api/v1/oauth/consent-dashboard");
      if (!res.ok) throw new Error("HTTP " + res.status);
      setData(await res.json());
    } catch (e: any) { setError(e.message); }
    finally { setLoading(false); }
  }, [baseUrl]);

  const revoke = useCallback(async (clientId: string) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(baseUrl + "/api/v1/oauth/consent-dashboard/" + clientId + "/revoke", { method: "POST" });
      if (!res.ok) throw new Error("HTTP " + res.status);
      return true;
    } catch (e: any) { setError(e.message); return false; }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { data, loading, error, fetchData, revoke };
}
