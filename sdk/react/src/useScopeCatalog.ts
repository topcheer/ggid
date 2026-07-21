import { useState, useCallback } from "react";

export interface ScopeDef {
  name: string;
  description: string;
  risk_level: "low" | "medium" | "high";
  created_by: string;
  usage_count: number;
  used_by_clients: string[];
  deprecated: boolean;
}

export function useScopeCatalog(baseUrl: string = "") {
  const [scopes, setScopes] = useState<ScopeDef[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchScopes = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(baseUrl + "/api/v1/oauth/scope-catalog");
      if (!res.ok) throw new Error("HTTP " + res.status);
      const data = await res.json(); setScopes(data.scopes || data || []);
    } catch (e: any) { setError(e.message); }
    finally { setLoading(false); }
  }, [baseUrl]);

  const addScope = useCallback(async (payload: { name: string; description: string; risk_level: string }) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(baseUrl + "/api/v1/oauth/scope-catalog", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(payload) });
      if (!res.ok) throw new Error("HTTP " + res.status);
      return true;
    } catch (e: any) { setError(e.message); return false; }
    finally { setLoading(false); }
  }, [baseUrl]);

  const deprecateScope = useCallback(async (name: string) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(baseUrl + "/api/v1/oauth/scope-catalog/" + name + "/deprecate", { method: "POST" });
      if (!res.ok) throw new Error("HTTP " + res.status);
      return true;
    } catch (e: any) { setError(e.message); return false; }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { scopes, loading, error, fetchScopes, addScope, deprecateScope };
}
