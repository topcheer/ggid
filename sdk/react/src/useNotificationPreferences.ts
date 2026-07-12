import { useState, useCallback } from "react";
export interface NotificationPrefs { matrix: Record<string, Record<string, boolean>>; quiet_hours: { enabled: boolean; start: string; end: string; timezone: string }; digest_frequency: string; per_user_override: { user_id: string; username: string; overrides: Record<string, string[]> }[]; }
export function useNotificationPreferences(baseUrl: string = "") {
  const [prefs, setPrefs] = useState<NotificationPrefs | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const fetchPrefs = useCallback(async () => { setLoading(true); setError(null); try { const res = await fetch(baseUrl + "/api/v1/notification/preferences"); if (!res.ok) throw new Error("HTTP " + res.status); setPrefs(await res.json()); } catch (e: any) { setError(e.message); } finally { setLoading(false); } }, [baseUrl]);
  const savePrefs = useCallback(async (p: NotificationPrefs) => { setLoading(true); setError(null); try { const res = await fetch(baseUrl + "/api/v1/notification/preferences", { method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(p) }); if (!res.ok) throw new Error("HTTP " + res.status); return true; } catch (e: any) { setError(e.message); return false; } finally { setLoading(false); } }, [baseUrl]);
  return { prefs, loading, error, fetchPrefs, savePrefs };
}
