import { useState, useCallback } from "react";

export interface ProvisioningEvent {
  id: string;
  timestamp: string;
  user: string;
  source: "SCIM" | "JIT" | "manual";
  action: "create" | "update" | "disable" | "delete";
  target_app: string;
  status: "success" | "failed";
  error_detail: string | null;
}

export function useProvisioningLog(baseUrl: string = "") {
  const [events, setEvents] = useState<ProvisioningEvent[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchEvents = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(baseUrl + "/api/v1/identity/provisioning-log");
      if (!res.ok) throw new Error("HTTP " + res.status);
      const data = await res.json(); setEvents(data.events || data || []);
    } catch (e: any) { setError(e.message); }
    finally { setLoading(false); }
  }, [baseUrl]);

  const retry = useCallback(async (id: string) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(baseUrl + "/api/v1/identity/provisioning-log/" + id + "/retry", { method: "POST" });
      if (!res.ok) throw new Error("HTTP " + res.status);
      return true;
    } catch (e: any) { setError(e.message); return false; }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { events, loading, error, fetchEvents, retry };
}
