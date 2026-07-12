import { useState, useCallback } from "react";

export interface CibaPerClient {
  client_id: string;
  client_name: string;
  delivery_mode: "poll" | "ping" | "push";
  max_polling_interval: number;
}

export interface CibaUsageStats {
  total_requests: number;
  successful: number;
  expired: number;
  denied: number;
  by_mode: { poll: number; ping: number; push: number };
}

export interface CibaConfig {
  enabled: boolean;
  binding_message: {
    required: boolean;
    max_chars: number;
    pattern: string;
  };
  max_polling_interval: number;
  token_delivery_mode: "poll" | "ping" | "push";
  per_client: CibaPerClient[];
  usage_stats: CibaUsageStats;
}

export function useCibaConfig(baseUrl: string = "") {
  const [config, setConfig] = useState<CibaConfig | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchConfig = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/ciba-config`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data: CibaConfig = await res.json();
      setConfig(data);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Unknown error");
    } finally {
      setLoading(false);
    }
  }, [baseUrl]);

  const updateConfig = useCallback(async (patch: Partial<CibaConfig>) => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/ciba-config`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(patch),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data: CibaConfig = await res.json();
      setConfig(data);
      return data;
    } catch (e) {
      setError(e instanceof Error ? e.message : "Unknown error");
      return null;
    } finally {
      setLoading(false);
    }
  }, [baseUrl]);

  return { config, loading, error, fetchConfig, updateConfig };
}
