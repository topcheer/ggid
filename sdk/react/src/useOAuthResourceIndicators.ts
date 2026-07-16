import { useState, useCallback, useEffect } from "react";

/**
 * DEMO DATA — Tries real API first, falls back to empty demo data.
 */

export interface ResourcePattern {
  match_type: "exact" | "wildcard" | "regex";
  pattern: string;
}

export interface ClientPatterns {
  client_id: string;
  patterns: ResourcePattern[];
}

export interface ScopeRestriction {
  scope: string;
  allowed_resources: string[];
  restricted: boolean;
}

export interface RejectedRequest {
  client: string;
  requested_resource: string;
  reason: string;
  timestamp: string;
}

export interface OAuthResourceIndicatorsData {
  resource_indicator_required: boolean;
  per_client_patterns: ClientPatterns[];
  per_scope_restriction: ScopeRestriction[];
  rejected_requests_log: RejectedRequest[];
}

export function useOAuthResourceIndicators() {
  const [data, setData] = useState<OAuthResourceIndicatorsData | null>(null);
  const [isDemoData, setIsDemoData] = useState(true);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      // Try real API first
      let res: Response | null = null;
      try { res = await fetch("/api/v1/data", { headers: { "Content-Type": "application/json" } }); } catch { res = null; }
      if (res?.ok) { const d = await res.json(); setData(d); setIsDemoData(false); return; }
      setIsDemoData(true);
      setData({
        resource_indicator_required: true,
        per_client_patterns: [
          { client_id: "client-web-001", patterns: [
            { match_type: "wildcard", pattern: "https://api.ggid.dev/v1/*" },
            { match_type: "exact", pattern: "https://auth.ggid.dev/userinfo" },
          ]},
          { client_id: "client-api-003", patterns: [
            { match_type: "regex", pattern: "^https://internal\\.ggid\\.dev/(v[23])/.*$" },
            { match_type: "wildcard", pattern: "https://audit.ggid.dev/v1/*" },
          ]},
        ],
        per_scope_restriction: [
          { scope: "openid", allowed_resources: ["/userinfo"], restricted: true },
          { scope: "read", allowed_resources: ["/v1/users", "/v1/orgs", "/v1/audit"], restricted: true },
          { scope: "admin", allowed_resources: ["*"], restricted: false },
        ],
        rejected_requests_log: [
          { client: "client-web-001", requested_resource: "https://evil.com/callback", reason: "Pattern not matched", timestamp: "2h ago" },
          { client: "client-mobile-002", requested_resource: "http://localhost:3000", reason: "HTTP not allowed", timestamp: "5h ago" },
        ],
      });
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load data");
    } finally {
      setLoading(false);
    }
  }, []);

  const testResource = useCallback((_resource: string) => {
    return _resource.includes("ggid.dev");
  }, []);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  return { data, loading, error, refresh: fetchData, testResource };
}
