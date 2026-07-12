import { useState, useCallback } from "react";

export interface Snapshot {
  total_users: number;
  by_status: { status: string; count: number }[];
  by_org: { org: string; count: number }[];
  by_role: { role: string; count: number }[];
  changes_24h: { created: number; deleted: number; modified: number; net: number };
  snapshot_at: string;
}

export function useDirectorySnapshot(baseUrl: string = "") {
  const [snapshot, setSnapshot] = useState<Snapshot | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchSnapshot = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/identity/directory-snapshot`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setSnapshot(await res.json());
    } catch (e: any) { setError(e.message); setSnapshot(null); }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { snapshot, loading, error, fetchSnapshot };
}
