import { useState, useCallback } from "react";
export interface ClientCreds { client_id: string; client_secret: string; }
export function useClientOnboarding(baseUrl: string = "") {
  const [creds, setCreds] = useState<ClientCreds | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const register = useCallback(async (payload: { name: string; description: string; grants: string[]; redirects: string[]; scopes: string[] }) => {
    setLoading(true); setError(null);
    try { const res = await fetch(baseUrl + "/api/v1/oauth/clients", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(payload) }); if (!res.ok) throw new Error("HTTP " + res.status); const data = await res.json(); setCreds(data); return data; } catch (e: any) { setError(e.message); return null; } finally { setLoading(false); }
  }, [baseUrl]);
  return { creds, loading, error, register };
}
