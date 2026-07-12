import { useState, useCallback } from "react";
export interface GrantEvent { id: string; client_name: string; user_id: string; username: string; scopes: string[]; granted_at: string; expires_at: string; revoked_at: string | null; grant_type: string; }
export function useGrantHistory(baseUrl: string = "") {
  const [events, setEvents] = useState<GrantEvent[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const fetchHistory = useCallback(async () => { setLoading(true); setError(null); try { const res = await fetch(baseUrl + "/api/v1/oauth/grant-history"); if (!res.ok) throw new Error("HTTP " + res.status); const data = await res.json(); setEvents(data.events || data || []); } catch (e: any) { setError(e.message); } finally { setLoading(false); } }, [baseUrl]);
  return { events, loading, error, fetchHistory };
}
