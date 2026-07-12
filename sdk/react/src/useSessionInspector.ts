import { useState, useCallback } from "react";
export interface SessionInfo { id: string; device: string; ip_address: string; location: string; created_at: string; last_active: string; mfa_verified: boolean; scopes: string[]; expires_at: string; }
export function useSessionInspector(baseUrl: string = "") {
  const [sessions, setSessions] = useState<SessionInfo[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const searchUser = useCallback(async (user: string) => { setLoading(true); setError(null); try { const res = await fetch(baseUrl + "/api/v1/auth/session-inspector?user=" + encodeURIComponent(user)); if (!res.ok) throw new Error("HTTP " + res.status); const data = await res.json(); setSessions(data.sessions || data || []); } catch (e: any) { setError(e.message); } finally { setLoading(false); } }, [baseUrl]);
  const revoke = useCallback(async (id: string) => { setLoading(true); setError(null); try { const res = await fetch(baseUrl + "/api/v1/auth/session-inspector/" + id, { method: "DELETE" }); if (!res.ok) throw new Error("HTTP " + res.status); return true; } catch (e: any) { setError(e.message); return false; } finally { setLoading(false); } }, [baseUrl]);
  return { sessions, loading, error, searchUser, revoke };
}
