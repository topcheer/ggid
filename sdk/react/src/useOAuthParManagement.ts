import { useState, useCallback, useEffect } from "react";

export interface PushedRequest {
  request_uri: string;
  client_id: string;
  client_name: string;
  pushed_at: string;
  expires_at: string;
  consumed: boolean;
}

export interface ParClientUsage {
  client_id: string;
  client_name: string;
  request_count: number;
}

export interface ParError {
  error_code: string;
  description: string;
  count: number;
}

export interface OAuthParManagementData {
  active_pushed_requests: PushedRequest[];
  par_cache_size: number;
  par_hit_rate: number;
  expired_cleanup_count: number;
  per_client_usage: ParClientUsage[];
  error_responses: ParError[];
}

export function useOAuthParManagement() {
  const [data, setData] = useState<OAuthParManagementData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      await new Promise((r) => setTimeout(r, 400));
      setData({
        active_pushed_requests: [
          { request_uri: "urn:ietf:params:oauth:request_uri:6ev3j4kq", client_id: "client-fin-001", client_name: "Finance Dashboard", pushed_at: "2m ago", expires_at: "58s left", consumed: false },
          { request_uri: "urn:ietf:params:oauth:request_uri:8mf9x2np", client_id: "client-mobile-002", client_name: "Mobile Banking App", pushed_at: "5m ago", expires_at: "55s left", consumed: false },
          { request_uri: "urn:ietf:params:oauth:request_uri:3kd7w1tr", client_id: "client-read-003", client_name: "Analytics Reader", pushed_at: "10m ago", expires_at: "Expired", consumed: true },
          { request_uri: "urn:ietf:params:oauth:request_uri:9pv2m6zb", client_id: "client-fin-001", client_name: "Finance Dashboard", pushed_at: "12m ago", expires_at: "Expired", consumed: true },
        ],
        par_cache_size: 1842,
        par_hit_rate: 94.2,
        expired_cleanup_count: 38,
        per_client_usage: [
          { client_id: "client-fin-001", client_name: "Finance Dashboard", request_count: 820 },
          { client_id: "client-mobile-002", client_name: "Mobile Banking App", request_count: 540 },
          { client_id: "client-read-003", client_name: "Analytics Reader", request_count: 310 },
          { client_id: "client-cli-004", client_name: "CLI Tool", request_count: 172 },
        ],
        error_responses: [
          { error_code: "invalid_request_uri", description: "Request URI not found or expired", count: 8 },
          { error_code: "invalid_request_object", description: "Malformed request object", count: 3 },
          { error_code: "expired_request", description: "Request object past TTL", count: 12 },
        ],
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
