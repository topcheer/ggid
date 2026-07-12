import { useState, useCallback } from "react";
export interface ScimConfig { endpoint_url: string; mappings: { scim_attr: string; local_attr: string }[]; rules: { create: boolean; update: boolean; deactivate: boolean }; sync_direction: string; last_sync: string; last_status: string; error_queue: { id: string; user: string; error: string; timestamp: string }[]; }
export function useScimProvisioning(baseUrl: string = "") {
  const [data, setData] = useState<ScimConfig | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const fetchData = useCallback(async () => { setLoading(true); setError(null); try { const res = await fetch(baseUrl + "/api/v1/identity/scim-provisioning"); if (!res.ok) throw new Error("HTTP " + res.status); setData(await res.json()); } catch (e: any) { setError(e.message); } finally { setLoading(false); } }, [baseUrl]);
  return { data, loading, error, fetchData };
}
