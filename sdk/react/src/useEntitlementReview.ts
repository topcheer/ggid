import { useState, useCallback } from "react";

export interface Entitlement {
  id: string;
  resource: string;
  action: string;
  source: "direct" | "inherited";
  via_group: string | null;
  last_used: string | null;
  unused_90d: boolean;
  over_privileged: boolean;
  recommendation: "keep" | "revoke" | "reduce";
}

export function useEntitlementReview(baseUrl: string = "") {
  const [permissions, setPermissions] = useState<Entitlement[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const reviewUser = useCallback(async (userId: string) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(baseUrl + "/api/v1/identity/entitlement-review?user=" + encodeURIComponent(userId));
      if (!res.ok) throw new Error("HTTP " + res.status);
      const data = await res.json(); setPermissions(data.permissions || data || []);
    } catch (e: any) { setError(e.message); }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { permissions, loading, error, reviewUser };
}
