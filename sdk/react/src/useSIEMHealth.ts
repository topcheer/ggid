import { useState, useCallback } from "react";

export interface SIEMDestination {
  id: string;
  name: string;
  endpoint: string;
  status: "healthy" | "degraded" | "down";
  connectivity: boolean;
  latency_ms: number;
  throughput_per_sec: number;
  error_rate: number;
  last_success: string;
}

export function useSIEMHealth(baseUrl: string = "") {
  const [destinations, setDestinations] = useState<SIEMDestination[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchHealth = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/audit/siem-health`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json(); setDestinations(data.destinations || data || []);
    } catch (e: any) { setError(e.message); }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { destinations, loading, error, fetchHealth };
}
