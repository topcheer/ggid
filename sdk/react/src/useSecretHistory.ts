import { useState, useCallback } from "react";

export interface RotationEntry {
  id: string;
  rotated_at: string;
  rotated_by: string;
  thumbprint: string;
  age_days: number;
}

export interface SecretHistory {
  client_id: string;
  client_name: string;
  current: RotationEntry;
  previous: RotationEntry | null;
  rotation_log: RotationEntry[];
}

export function useSecretHistory(baseUrl: string = "") {
  const [data, setData] = useState<SecretHistory | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchHistory = useCallback(async (clientId: string) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/oauth/secret-history?client_id=${encodeURIComponent(clientId)}`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setData(await res.json());
    } catch (e: any) { setError(e.message); }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { data, loading, error, fetchHistory };
}
