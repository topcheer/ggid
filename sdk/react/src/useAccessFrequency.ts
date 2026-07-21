import { useState, useCallback } from "react";

export interface HourlyBucket {
  hour: string;
  access_count: number;
  unique_users: number;
  is_anomaly: boolean;
}

export interface FrequencyData {
  resource_id: string;
  resource_name: string;
  buckets: HourlyBucket[];
  total_accesses: number;
  avg_per_hour: number;
  anomaly_count: number;
  peak_hour: string;
  peak_count: number;
}

export function useAccessFrequency(baseUrl: string = "") {
  const [data, setData] = useState<FrequencyData | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchFrequency = useCallback(async (resourceId: string) => {
    if (!resourceId) return;
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/policy/access-frequency?resource=${encodeURIComponent(resourceId)}`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const json: FrequencyData = await res.json();
      setData(json);
    } catch (e: any) {
      setError(e.message);
      setData(null);
    } finally {
      setLoading(false);
    }
  }, [baseUrl]);

  return { data, loading, error, fetchFrequency };
}
