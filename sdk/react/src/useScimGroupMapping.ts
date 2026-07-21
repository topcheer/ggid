import { useState, useCallback } from "react";
export interface GroupMap { id: string; external_group: string; local_role: string; auto_provision: boolean; sync_direction: string; last_sync: string; last_status: string; }
export function useScimGroupMapping(baseUrl: string = "") {
  const [mappings, setMappings] = useState<GroupMap[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const fetchMappings = useCallback(async () => { setLoading(true); setError(null); try { const res = await fetch(baseUrl + "/api/v1/identity/scim-group-mapping"); if (!res.ok) throw new Error("HTTP " + res.status); const d = await res.json(); setMappings(d.mappings || d || []); } catch (e: any) { setError(e.message); } finally { setLoading(false); } }, [baseUrl]);
  return { mappings, loading, error, fetchMappings };
}
