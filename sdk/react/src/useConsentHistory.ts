import { useState, useCallback } from "react";

export interface ConsentEntry {
  id: string;
  action: "granted" | "revoked";
  user_id: string;
  username: string;
  client_id: string;
  client_name: string;
  scopes: string[];
  timestamp: string;
  ip_address: string;
}

export function useConsentHistory(baseUrl: string = "") {
  const [entries, setEntries] = useState<ConsentEntry[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchHistory = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/oauth/consent-history`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json(); setEntries(data.entries || data || []);
    } catch (e: any) { setError(e.message); }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { entries, loading, error, fetchHistory };
}
