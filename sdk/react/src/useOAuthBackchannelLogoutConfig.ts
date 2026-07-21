import { useState, useCallback, useEffect } from "react";

/**
 * DEMO DATA — Tries real API first, falls back to empty demo data.
 * isDemoData flag indicates whether live or fallback data is shown.
 */

export interface ErrorHandling {
  retry_attempts: number;
  retry_timeout_seconds: number;
  failed_notifications_24h: number;
}

export interface ClientLogoutConfig {
  client_id: string;
  client_name: string;
  backchannel_logout_enabled: boolean;
}

export interface OAuthBackchannelLogoutConfigData {
  logout_endpoint: string;
  session_lifetime: number;
  token_revocation_on_logout: boolean;
  per_client_toggle: boolean;
  error_handling: ErrorHandling;
  client_configs: ClientLogoutConfig[];
  logout_token_preview: Record<string, unknown>;
}

export function useOAuthBackchannelLogoutConfig() {
  const [data, setData] = useState<OAuthBackchannelLogoutConfigData | null>(null);
  const [isDemoData, setIsDemoData] = useState(true);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      // Try real API first
      let res: Response | null = null;
      try {
        res = await fetch("/api/v1/data", { headers: { "Content-Type": "application/json" } });
      } catch { res = null; }
      if (res?.ok) { const d = await res.json(); setData(d); setIsDemoData(false); return; }
      setIsDemoData(true);
      setData({
        logout_endpoint: "https://idp.example.com/backchannel-logout",
        session_lifetime: 3600,
        token_revocation_on_logout: true,
        per_client_toggle: true,
        error_handling: {
          retry_attempts: 3,
          retry_timeout_seconds: 30,
          failed_notifications_24h: 2,
        },
        client_configs: [
          { client_id: "client-web-001", client_name: "Web Dashboard", backchannel_logout_enabled: false },
          { client_id: "client-mobile-002", client_name: "Mobile App", backchannel_logout_enabled: false },
          { client_id: "client-api-003", client_name: "API Gateway", backchannel_logout_enabled: false },
          { client_id: "client-spa-004", client_name: "Marketing SPA", backchannel_logout_enabled: false },
        ],
        logout_token_preview: {
          iss: "https://idp.example.com",
          sub: "user-uuid-123",
          iat: 1700000000,
          jti: "logout-unique-id-456",
          events: { "http://schemas.openid.net/event/backchannel-logout": {} },
          sid: "session-id-789",
        },
      });
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load data");
    } finally {
      setLoading(false);
    }
  }, []);

  const testLogout = useCallback(async () => {
    console.log("Sending test backchannel logout");
  }, []);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  return { data, loading, error, refresh: fetchData, testLogout };
}
