import { useState, useCallback } from "react";

export interface DiscoveryInfo {
  issuer: string;
  well_known_url: string;
  supported_scopes: { name: string; description: string }[];
  supported_grants: string[];
  signing_algs: string[];
  userinfo_endpoint: string;
  userinfo_status: "up" | "down";
  jwks_uri: string;
  jwks_last_refresh: string;
  jwks_key_count: number;
}

export function useDiscoveryConfig(baseUrl: string = "") {
  const [data, setData] = useState<DiscoveryInfo | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(baseUrl + "/api/v1/oauth/discovery-config");
      if (!res.ok) throw new Error("HTTP " + res.status);
      setData(await res.json());
    } catch (e: any) { setError(e.message); }
    finally { setLoading(false); }
  }, [baseUrl]);

  const refreshJwks = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(baseUrl + "/api/v1/oauth/discovery-config/jwks-refresh", { method: "POST" });
      if (!res.ok) throw new Error("HTTP " + res.status);
      return true;
    } catch (e: any) { setError(e.message); return false; }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { data, loading, error, fetchData, refreshJwks };
}
