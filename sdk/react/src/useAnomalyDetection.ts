import { useState, useCallback } from "react";

export interface AnomalyEvent {
  id: string;
  type: string;
  severity: "low" | "medium" | "high" | "critical";
  user: string;
  timestamp: string;
  confidence: number;
  detail: string;
  status: "active" | "acknowledged" | "dismissed";
}

export function useAnomalyDetection(baseUrl: string = "") {
  const [events, setEvents] = useState<AnomalyEvent[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchEvents = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(baseUrl + "/api/v1/audit/anomaly-detection");
      if (!res.ok) throw new Error("HTTP " + res.status);
      const data = await res.json(); setEvents(data.events || data || []);
    } catch (e: any) { setError(e.message); }
    finally { setLoading(false); }
  }, [baseUrl]);

  const updateStatus = useCallback(async (id: string, status: string) => {
    try { await fetch(baseUrl + "/api/v1/audit/anomaly-detection/" + id, { method: "PATCH", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ status }) }); }
    catch { /* noop */ }
  }, [baseUrl]);

  return { events, loading, error, fetchEvents, updateStatus };
}
