import { useState, useCallback, useEffect } from "react";

export interface RateLimitEntry {
  client_id: string;
  requests_per_min: number;
  burst: number;
  concurrent_tokens: number;
  daily_quota: number;
  current_usage_today: number;
}

export interface EndpointOverride {
  client_id: string;
  endpoint: string;
  override_req_per_min: number | null;
  override_burst: number | null;
}

export interface ThrottleResponse {
  status_code: number;
  retry_after_seconds: number;
}

export interface OAuthRateLimitPerClientData {
  rate_limits: RateLimitEntry[];
  per_endpoint_override: EndpointOverride[];
  throttle_response: ThrottleResponse;
  whitelist_ips: string[];
}

export function useOAuthRateLimitPerClient() {
  const [data, setData] = useState<OAuthRateLimitPerClientData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      await new Promise((r) => setTimeout(r, 400));
      setData({
        rate_limits: [
          { client_id: "client-web-001", requests_per_min: 100, burst: 20, concurrent_tokens: 50, daily_quota: 50000, current_usage_today: 12400 },
          { client_id: "client-mobile-002", requests_per_min: 200, burst: 50, concurrent_tokens: 100, daily_quota: 100000, current_usage_today: 89000 },
          { client_id: "client-api-003", requests_per_min: 500, burst: 100, concurrent_tokens: 200, daily_quota: 500000, current_usage_today: 342000 },
          { client_id: "client-spa-005", requests_per_min: 60, burst: 10, concurrent_tokens: 30, daily_quota: 20000, current_usage_today: 5600 },
        ],
        per_endpoint_override: [
          { client_id: "client-web-001", endpoint: "/oauth/authorize", override_req_per_min: 30, override_burst: 5 },
          { client_id: "client-web-001", endpoint: "/oauth/token", override_req_per_min: 20, override_burst: 3 },
          { client_id: "client-api-003", endpoint: "/oauth/introspect", override_req_per_min: 1000, override_burst: 200 },
        ],
        throttle_response: { status_code: 429, retry_after_seconds: 60 },
        whitelist_ips: ["10.0.0.1", "10.0.0.2"],
      });
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load data");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  return { data, loading, error, refresh: fetchData };
}
