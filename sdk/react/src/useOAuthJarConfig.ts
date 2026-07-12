import { useState, useCallback, useEffect } from "react";

export interface ClientOverride {
  client_id: string;
  jar_required: boolean;
  signing_alg: string;
  lifetime_seconds: number;
}

export interface JarUsageStats {
  total_requests_24h: number;
  jar_requests_24h: number;
  adoption_rate_pct: number;
  validation_failures_24h: number;
}

export interface OAuthJarConfigData {
  require_jar: boolean;
  jar_lifetime_seconds: number;
  signing_alg: string;
  encryption_optional: boolean;
  per_client_override: ClientOverride[];
  request_object_preview: Record<string, unknown>;
  jar_usage_stats: JarUsageStats;
}

export function useOAuthJarConfig() {
  const [data, setData] = useState<OAuthJarConfigData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      await new Promise((r) => setTimeout(r, 400));
      setData({
        require_jar: true,
        jar_lifetime_seconds: 300,
        signing_alg: "RS256",
        encryption_optional: true,
        per_client_override: [
          { client_id: "client-web-001", jar_required: false, signing_alg: "RS256", lifetime_seconds: 600 },
          { client_id: "client-mobile-002", jar_required: true, signing_alg: "ES256", lifetime_seconds: 300 },
          { client_id: "client-api-003", jar_required: true, signing_alg: "PS256", lifetime_seconds: 180 },
        ],
        request_object_preview: {
          iss: "client-mobile-002",
          aud: "https://auth.ggid.dev",
          response_type: "code",
          client_id: "client-mobile-002",
          redirect_uri: "myapp://callback",
          scope: "openid profile",
          state: "xyz123",
          nonce: "abc456",
          exp: 1735341600,
          iat: 1735341300,
        },
        jar_usage_stats: {
          total_requests_24h: 15420,
          jar_requests_24h: 11200,
          adoption_rate_pct: 73,
          validation_failures_24h: 4,
        },
      });
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load data");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);
  return { data, loading, error, refresh: fetchData };
}
