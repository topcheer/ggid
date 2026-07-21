import { useState, useCallback } from "react";

export interface FederatedIdP {
  id: string;
  provider_type: "saml" | "oidc";
  entity_id: string;
  name: string;
  status: "active" | "inactive" | "error";
  last_sync: string;
  trust_level: "full" | "limited" | "conditional";
}

export function useIdpFederation(baseUrl: string = "") {
  const [idps, setIdps] = useState<FederatedIdP[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchIdps = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(baseUrl + "/api/v1/identity/idp-federation");
      if (!res.ok) throw new Error("HTTP " + res.status);
      const data = await res.json(); setIdps(data.idps || data || []);
    } catch (e: any) { setError(e.message); }
    finally { setLoading(false); }
  }, [baseUrl]);

  const addIdp = useCallback(async (payload: { provider_type: string; entity_id: string; name: string; trust_level: string }) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(baseUrl + "/api/v1/identity/idp-federation", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(payload) });
      if (!res.ok) throw new Error("HTTP " + res.status);
      return true;
    } catch (e: any) { setError(e.message); return false; }
    finally { setLoading(false); }
  }, [baseUrl]);

  const testConnection = useCallback(async (id: string) => {
    try { await fetch(baseUrl + "/api/v1/identity/idp-federation/" + id + "/test", { method: "POST" }); return true; }
    catch { return false; }
  }, [baseUrl]);

  return { idps, loading, error, fetchIdps, addIdp, testConnection };
}
