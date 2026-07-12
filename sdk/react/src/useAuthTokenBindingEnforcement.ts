import { useState, useCallback, useEffect } from "react";

export interface ClientBindingPolicy {
  client_id: string;
  min_binding_strength: string;
  allowed_methods: string[];
  non_compliant_count: number;
}

export interface MigrationPhase {
  phase: string;
  description: string;
  status: string;
}

export interface AuthTokenBindingEnforcementData {
  enforcement_level: string;
  per_client_binding_policy: ClientBindingPolicy[];
  grace_period_days: number;
  non_compliant_tokens_count: number;
  auto_revoke_enabled: boolean;
  migration_timeline: MigrationPhase[];
}

export function useAuthTokenBindingEnforcement() {
  const [data, setData] = useState<AuthTokenBindingEnforcementData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      await new Promise((r) => setTimeout(r, 400));
      setData({
        enforcement_level: "required",
        per_client_binding_policy: [
          { client_id: "client-banking-app", min_binding_strength: "strict", allowed_methods: ["mTLS", "DPoP"], non_compliant_count: 0 },
          { client_id: "client-mobile-002", min_binding_strength: "required", allowed_methods: ["DPoP"], non_compliant_count: 3 },
          { client_id: "client-internal-cli", min_binding_strength: "optional", allowed_methods: ["DPoP", "PKI"], non_compliant_count: 0 },
          { client_id: "client-legacy-svc", min_binding_strength: "none", allowed_methods: [], non_compliant_count: 12 },
        ],
        grace_period_days: 30,
        non_compliant_tokens_count: 15,
        auto_revoke_enabled: true,
        migration_timeline: [
          { phase: "Discovery", description: "Identify all clients and their current binding status", status: "completed" },
          { phase: "Notification", description: "Notify non-compliant client owners", status: "completed" },
          { phase: "Optional Phase", description: "Binding supported but not required", status: "active" },
          { phase: "Required Phase", description: "Reject tokens without binding", status: "pending" },
          { phase: "Strict Phase", description: "Enforce highest binding strength only", status: "pending" },
        ],
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
