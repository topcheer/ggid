import { useState, useCallback, useEffect } from "react";

/**
 * DEMO DATA — Tries real API first, falls back to empty demo data.
 */

export interface SamlMapping {
  id: string;
  source_attribute: string;
  target_field: string;
  transform_rule: string;
}

export interface IdpOverride {
  idp: string;
  override_count: number;
  status: string;
}

export interface SamlAttributeMappingConfigData {
  mappings: SamlMapping[];
  per_idp: IdpOverride[];
}

export function useSamlAttributeMappingConfig() {
  const [data, setData] = useState<SamlAttributeMappingConfigData | null>(null);
  const [isDemoData, setIsDemoData] = useState(true);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const testMapping = useCallback(async (_id: string) => { /* mock */ }, []);
  const fetchData = useCallback(async () => {
    setLoading(true); setError(null);
    try { // Try real API first
      let res: Response | null = null;
      try { res = await fetch("/api/v1/data", { headers: { "Content-Type": "application/json" } }); } catch { res = null; }
      if (res?.ok) { const d = await res.json(); setData(d); setIsDemoData(false); return; }
      setIsDemoData(true);
      setData({ mappings: [
        { id: "m1", source_attribute: "http://schemas.../emailaddress", target_field: "email", transform_rule: "direct" },
        { id: "m2", source_attribute: "http://schemas.../givenname", target_field: "first_name", transform_rule: "direct" },
        { id: "m3", source_attribute: "http://schemas.../groups", target_field: "groups", transform_rule: "regex:CN=([^,]+)" },
      ], per_idp: [
        { idp: "Azure AD", override_count: 2, status: "active" },
        { idp: "Okta", override_count: 0, status: "default" },
      ] });
    } catch (e) { setError(e instanceof Error ? e.message : "Failed"); } finally { setLoading(false); }
  }, []);
  useEffect(() => { fetchData(); }, [fetchData]);
  return { data, loading, error, refresh: fetchData, testMapping };
}
