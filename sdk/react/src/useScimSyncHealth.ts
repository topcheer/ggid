import { useState, useCallback } from "react";

export interface ScimHealth {
  endpoint_url: string;
  last_sync_at: string;
  provisioning_errors: { timestamp: string; user_id: string; error: string }[];
  user_counts: { synced: number; pending: number; failed: number };
  rate_limit: { remaining: number; reset_at: string };
  throughput_per_min: number;
  status: "healthy" | "degraded" | "error";
}

export function useScimSyncHealth(baseUrl: string = "") {
  const [data, setData] = useState<ScimHealth | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchHealth = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/identity/scim-sync-health`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setData(await res.json());
    } catch (e: any) { setError(e.message); }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { data, loading, error, fetchHealth };
}
