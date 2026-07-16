import { useState, useCallback, useEffect } from "react";

/**
 * DEMO DATA — Tries real API first, falls back to empty demo data.
 * isDemoData flag indicates whether live or fallback data is shown.
 */

export interface ParRequest {
  request_uri: string;
  client: string;
  pushed_at: string;
  expires_at: string;
  consumed: boolean;
}

export interface ParClientUsageEntry {
  client: string;
  count: number;
  pct: number;
}

export interface ParErrorEntry {
  error: string;
  count: number;
}

export interface OAuthParUsageData {
  total_pushed: number;
  active_requests: ParRequest[];
  cache_size: number;
  hit_rate: number;
  per_client: ParClientUsageEntry[];
  errors: ParErrorEntry[];
}

export function useOAuthParUsage() {
  const [data, setData] = useState<OAuthParUsageData | null>(null);
  const [isDemoData, setIsDemoData] = useState(true);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const fetchData = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      // Try real API first
      let res: Response | null = null;
      try {
        res = await fetch("/api/v1/data", { headers: { "Content-Type": "application/json" } });
      } catch { res = null; }
      if (res?.ok) { const d = await res.json(); setData(d); setIsDemoData(false); return; }
      setIsDemoData(true);
      setData({
        total_pushed: 18500,
        active_requests: [
          { request_uri: "urn:ietf:params:oauth:request_uri:abc123", client: "web-console", pushed_at: "1m ago", expires_at: "59s", consumed: false },
          { request_uri: "urn:ietf:params:oauth:request_uri:def456", client: "mobile-app", pushed_at: "30s ago", expires_at: "30s", consumed: false },
          { request_uri: "urn:ietf:params:oauth:request_uri:ghi789", client: "ci-cd-bot", pushed_at: "55s ago", expires_at: "5s", consumed: true },
        ],
        cache_size: 42,
        hit_rate: 94,
        per_client: [
          { client: "web-console", count: 8200, pct: 44 },
          { client: "mobile-app", count: 5100, pct: 28 },
          { client: "ci-cd-bot", count: 3200, pct: 17 },
          { client: "partner-api", count: 2000, pct: 11 },
        ],
        errors: [
          { error: "invalid_request_uri", count: 45 },
          { error: "expired_request", count: 28 },
          { error: "client_mismatch", count: 12 },
        ],
      });
    } catch (e) { setError(e instanceof Error ? e.message : "Failed"); }
    finally { setLoading(false); }
  }, []);
  useEffect(() => { fetchData(); }, [fetchData]);
  return { data, loading, error, refresh: fetchData, isDemoData };
}
