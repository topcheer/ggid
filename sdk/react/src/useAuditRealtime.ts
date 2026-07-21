import { useState, useCallback } from "react";

export interface LiveEvent {
  id: string;
  timestamp: string;
  user: string;
  action: string;
  resource: string;
  ip_address: string;
  result: "success" | "denied" | "error";
  severity: "info" | "warning" | "error" | "critical";
}

export function useAuditRealtime(baseUrl: string = "") {
  const [events, setEvents] = useState<LiveEvent[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchEvents = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(baseUrl + "/api/v1/audit/realtime?limit=50");
      if (!res.ok) throw new Error("HTTP " + res.status);
      const data = await res.json(); setEvents((data.events || data || []).slice(0, 50));
    } catch (e: any) { setError(e.message); }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { events, loading, error, fetchEvents, setEvents };
}
