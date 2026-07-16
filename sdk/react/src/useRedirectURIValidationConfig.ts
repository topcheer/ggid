import { useState, useCallback, useEffect } from "react";

/**
 * DEMO DATA — Tries real API first, falls back to empty demo data.
 */

export interface ClientUriEntry {
  client: string;
  uris: string[];
}

export interface RedirectURIValidationConfigData {
  https_only: boolean;
  exact_match_only: boolean;
  localhost_allowlist: boolean;
  fragment_allowed: boolean;
  custom_schemes: string[];
  per_client: ClientUriEntry[];
}

export function useRedirectURIValidationConfig() {
  const [data, setData] = useState<RedirectURIValidationConfigData | null>(null);
  const [isDemoData, setIsDemoData] = useState(true);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const testUri = useCallback(async (_uri: string) => { /* mock */ }, []);
  const fetchData = useCallback(async () => {
    setLoading(true); setError(null);
    try { // Try real API first
      let res: Response | null = null;
      try { res = await fetch("/api/v1/data", { headers: { "Content-Type": "application/json" } }); } catch { res = null; }
      if (res?.ok) { const d = await res.json(); setData(d); setIsDemoData(false); return; }
      setIsDemoData(true);
      setData({ https_only: true, exact_match_only: true, localhost_allowlist: true, fragment_allowed: false,
        custom_schemes: ["myapp://", "com.example.app:/oauth2redirect"],
        per_client: [{ client: "web-console", uris: ["https://console.ggid.dev/auth/callback"] }, { client: "mobile-app", uris: ["myapp://oauth/callback", "https://app.ggid.dev/callback"] }], });
    } catch (e) { setError(e instanceof Error ? e.message : "Failed"); } finally { setLoading(false); }
  }, []);
  useEffect(() => { fetchData(); }, [fetchData]);
  return { data, loading, error, refresh: fetchData, testUri };
}
