import { useState, useCallback } from "react";

export interface ClientEvent {
  id: string;
  event_type: "created" | "updated" | "rotated" | "suspended" | "reinstated" | "deleted";
  actor: string;
  timestamp: string;
  details: string;
  metadata: Record<string, string>;
}

export function useClientEvents(baseUrl: string = "") {
  const [events, setEvents] = useState<ClientEvent[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchEvents = useCallback(async (clientId: string) => {
    if (!clientId) return;
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/oauth/clients/${clientId}/events`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json();
      setEvents(data.events || data || []);
    } catch (e: any) {
      setError(e.message);
    } finally {
      setLoading(false);
    }
  }, [baseUrl]);

  return { events, loading, error, fetchEvents };
}
