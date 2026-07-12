import { useState, useCallback } from "react";

export interface TrustLifecycleRule {
  name: string;
  description: string;
  auto_revoke_after_days: number;
}

export interface MemberProvider {
  entity_id: string;
  name: string;
  status: "active" | "inactive";
  joined: string;
}

export interface FederationPatternsConfig {
  pattern: "hub_spoke" | "bilateral" | "multi_party";
  trust_lifecycle_rules: TrustLifecycleRule[];
  discovery_method: "metadata_url" | "webfinger" | "dns" | "manual";
  attribute_mapping_policy: string;
  slo_propagation: boolean;
  member_providers: MemberProvider[];
}

export function useFederationPatternsConfig(baseUrl: string = "") {
  const [config, setConfig] = useState<FederationPatternsConfig | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchConfig = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/federation-patterns-config`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setConfig(await res.json());
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); }
    finally { setLoading(false); }
  }, [baseUrl]);

  const updateConfig = useCallback(async (patch: Partial<FederationPatternsConfig>) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/federation-patterns-config`, {
        method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(patch),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json(); setConfig(data); return data;
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); return null; }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { config, loading, error, fetchConfig, updateConfig };
}
