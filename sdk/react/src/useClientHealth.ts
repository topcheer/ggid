import { useState, useCallback } from "react";

export interface ClientHealth {
  client_id: string;
  client_name: string;
  status: "healthy" | "warning" | "critical";
  active_tokens: number;
  error_rate: number;
  secret_expires: string | null;
  cert_expires: string | null;
  last_error: string | null;
}

export function useClientHealth(baseUrl: string = "") {
  const [clients, setClients] = useState<ClientHealth[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchHealth = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/oauth/client-health`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json(); setClients(data.clients || data || []);
    } catch (e: any) { setError(e.message); }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { clients, loading, error, fetchHealth };
}
