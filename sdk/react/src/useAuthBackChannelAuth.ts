import { useState, useCallback, useEffect } from "react";

export interface BindingMessageConfig {
  required: boolean;
  max_length: number;
  format_pattern: string;
}

export interface ClientCiba {
  client_id: string;
  delivery_mode: string;
  enabled: boolean;
}

export interface CibaUsageStats {
  ciba_requests_24h: number;
  successful_24h: number;
  rejected_24h: number;
  timeouts_24h: number;
}

export interface AuthBackChannelAuthData {
  enabled: boolean;
  binding_message_config: BindingMessageConfig;
  max_polling_interval_seconds: number;
  requested_expiry_max_seconds: number;
  token_delivery_mode: string;
  per_client_ciba: ClientCiba[];
  usage_stats: CibaUsageStats;
}

export function useAuthBackChannelAuth() {
  const [data, setData] = useState<AuthBackChannelAuthData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      await new Promise((r) => setTimeout(r, 400));
      setData({
        enabled: true,
        binding_message_config: { required: true, max_length: 128, format_pattern: "^[a-zA-Z0-9 ]+$" },
        max_polling_interval_seconds: 10,
        requested_expiry_max_seconds: 1200,
        token_delivery_mode: "poll",
        per_client_ciba: [
          { client_id: "client-banking-app", delivery_mode: "ping", enabled: true },
          { client_id: "client-kiosk-001", delivery_mode: "poll", enabled: true },
          { client_id: "client-mobile-002", delivery_mode: "push", enabled: false },
        ],
        usage_stats: { ciba_requests_24h: 342, successful_24h: 298, rejected_24h: 24, timeouts_24h: 20 },
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
