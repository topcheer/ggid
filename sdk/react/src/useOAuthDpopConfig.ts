import { useState, useCallback, useEffect } from "react";

export interface PerClientOverride {
  client_id: string;
  client_name: string;
  dpop_required: boolean;
}

export interface ExemptedClient {
  client_id: string;
  client_name: string;
}

export interface DpopStats {
  tokens_bound_24h: number;
  proofs_validated_24h: number;
  proofs_rejected_24h: number;
  replay_blocked: number;
  avg_latency_ms: number;
  non_confidential_clients: number;
}

export interface OAuthDpopConfigData {
  require_dpop: boolean;
  proof_max_age_seconds: number;
  key_binding_algorithm: string;
  dpop_stats: DpopStats;
  per_client_overrides: PerClientOverride[];
  exempted_clients: ExemptedClient[];
}

export function useOAuthDpopConfig() {
  const [data, setData] = useState<OAuthDpopConfigData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      await new Promise((r) => setTimeout(r, 400));
      setData({
        require_dpop: false,
        proof_max_age_seconds: 60,
        key_binding_algorithm: "ES256",
        dpop_stats: {
          tokens_bound_24h: 1842,
          proofs_validated_24h: 5210,
          proofs_rejected_24h: 38,
          replay_blocked: 4,
          avg_latency_ms: 12,
          non_confidential_clients: 3,
        },
        per_client_overrides: [
          { client_id: "client-fin-001", client_name: "Finance Dashboard", dpop_required: true },
          { client_id: "client-mobile-002", client_name: "Mobile Banking App", dpop_required: true },
          { client_id: "client-read-003", client_name: "Analytics Reader", dpop_required: false },
        ],
        exempted_clients: [
          { client_id: "client-legacy-099", client_name: "Legacy SOAP Integration" },
        ],
      });
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load data");
    } finally {
      setLoading(false);
    }
  }, []);

  const toggleRequireDpop = useCallback(async (enabled: boolean) => {
    setData((prev) => (prev ? { ...prev, require_dpop: enabled } : prev));
  }, []);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  return { data, loading, error, refresh: fetchData, toggleRequireDpop };
}
