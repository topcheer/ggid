import { useState, useCallback } from "react";

export interface TokenFamily {
  family_id: string;
  root_token: string;
  root_client: string;
  child_tokens: { id: string; client: string; issued_at: string; status: "active" | "revoked" }[];
  status: "active" | "revoked";
  reuse_detected: boolean;
}

export function useTokenFamily(baseUrl: string = "") {
  const [families, setFamilies] = useState<TokenFamily[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchFamilies = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(baseUrl + "/api/v1/oauth/token-family");
      if (!res.ok) throw new Error("HTTP " + res.status);
      const data = await res.json(); setFamilies(data.families || data || []);
    } catch (e: any) { setError(e.message); }
    finally { setLoading(false); }
  }, [baseUrl]);

  const revokeFamily = useCallback(async (familyId: string) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(baseUrl + "/api/v1/oauth/token-family/" + familyId + "/revoke", { method: "POST" });
      if (!res.ok) throw new Error("HTTP " + res.status);
      return true;
    } catch (e: any) { setError(e.message); return false; }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { families, loading, error, fetchFamilies, revokeFamily };
}
