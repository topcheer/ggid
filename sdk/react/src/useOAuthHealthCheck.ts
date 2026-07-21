import { useState, useCallback } from "react";

export interface EndpointStatus {
  name: string;
  url: string;
  status: "up" | "down" | "degraded";
  response_time_ms: number;
  last_check: string;
}

export interface OAuthHealthData {
  endpoints: EndpointStatus[];
  cert_expiry_days: number;
  last_check: string;
  failover_status: "active" | "standby" | "failed";
}

export function useOAuthHealthCheck(baseUrl: string = "") {
  const [data, setData] = useState<OAuthHealthData | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchHealth = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(baseUrl + "/api/v1/oauth/health-check");
      if (!res.ok) throw new Error("HTTP " + res.status);
      setData(await res.json());
    } catch (e: any) { setError(e.message); }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { data, loading, error, fetchHealth };
}
