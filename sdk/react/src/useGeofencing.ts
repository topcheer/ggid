import { useState, useCallback } from "react";

export interface GeofenceRule {
  allowed_countries: string[];
  denied_regions: string[];
  action: "allow" | "deny" | "mfa";
  enabled: boolean;
}

export function useGeofencing(baseUrl: string = "") {
  const [rule, setRule] = useState<GeofenceRule | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchRule = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/auth/geofencing`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data: GeofenceRule = await res.json();
      setRule(data);
    } catch (e: any) {
      setError(e.message);
    } finally {
      setLoading(false);
    }
  }, [baseUrl]);

  const saveRule = useCallback(async (newRule: GeofenceRule) => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/auth/geofencing`, { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(newRule) });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setRule(newRule);
      return true;
    } catch (e: any) {
      setError(e.message);
      return false;
    } finally {
      setLoading(false);
    }
  }, [baseUrl]);

  return { rule, loading, error, fetchRule, saveRule };
}
