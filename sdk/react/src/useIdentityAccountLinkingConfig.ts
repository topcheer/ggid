import { useState, useCallback, useEffect } from "react";

export interface LinkingMethod {
  method: "email_match" | "saml_subject" | "oidc_sub" | "manual";
  description: string;
  enabled: boolean;
}

export interface ConflictResolution {
  strategy: "manual" | "keep_oldest" | "merge";
  description: string;
}

export interface LinkedAccountsStats {
  total_linked: number;
  auto_linked_24h: number;
  conflicts_24h: number;
  unlinked_24h: number;
}

export interface UnlinkPolicy {
  allow_self_service: boolean;
  grace_period_hours: number;
  require_admin_approval: boolean;
}

export interface IdentityAccountLinkingConfigData {
  linking_methods: LinkingMethod[];
  auto_link_threshold: number;
  conflict_resolution: ConflictResolution;
  linked_accounts_stats: LinkedAccountsStats;
  unlink_policy: UnlinkPolicy;
  require_verification: boolean;
}

export function useIdentityAccountLinkingConfig() {
  const [data, setData] = useState<IdentityAccountLinkingConfigData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      await new Promise((r) => setTimeout(r, 400));
      setData({
        linking_methods: [
          { method: "email_match", description: "Match accounts by verified email address", enabled: true },
          { method: "saml_subject", description: "Match by SAML NameID subject ID", enabled: true },
          { method: "oidc_sub", description: "Match by OIDC subject identifier", enabled: true },
          { method: "manual", description: "Require manual admin approval for linking", enabled: false },
        ],
        auto_link_threshold: 0.9,
        conflict_resolution: {
          strategy: "keep_oldest",
          description: "When two accounts conflict, keep the oldest account and merge attributes.",
        },
        linked_accounts_stats: {
          total_linked: 12450,
          auto_linked_24h: 42,
          conflicts_24h: 3,
          unlinked_24h: 5,
        },
        unlink_policy: {
          allow_self_service: true,
          grace_period_hours: 48,
          require_admin_approval: false,
        },
        require_verification: true,
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
