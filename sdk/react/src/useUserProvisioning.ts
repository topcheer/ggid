import { useState, useCallback } from "react";
export interface ProvRule { id: string; source: string; trigger: string; action: string; target_app: string; enabled: boolean; }
export interface QueueItem { id: string; user: string; app: string; status: string; error: string | null; }
export function useUserProvisioning(baseUrl: string = "") {
  const [rules, setRules] = useState<ProvRule[]>([]);
  const [queue, setQueue] = useState<QueueItem[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const fetchData = useCallback(async () => { setLoading(true); setError(null); try { const res = await fetch(baseUrl + "/api/v1/identity/provisioning"); if (!res.ok) throw new Error("HTTP " + res.status); const d = await res.json(); setRules(d.rules || []); setQueue(d.queue || []); } catch (e: any) { setError(e.message); } finally { setLoading(false); } }, [baseUrl]);
  return { rules, queue, loading, error, fetchData };
}
