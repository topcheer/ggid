import { useState, useCallback } from "react";

export interface TrustAnchor {
  issuer: string;
  jwks_uri: string;
  trust_mark: string;
}

export interface EntityCategoryRequirement {
  category: string;
  required_claims: string[];
}

export interface FederatedProvider {
  issuer: string;
  name: string;
  status: "active" | "inactive";
}

export interface OidcFederationConfig {
  trust_anchors: TrustAnchor[];
  federated_providers: { issuer: string; name: string; status: "active" | "inactive" }[];
  auto_discovery: boolean;
  trust_resolution_policy: "tree" | "path" | "graph";
  entity_category_requirements: EntityCategoryRequirement[];
}

export function useOidcFederationConfig(baseUrl: string = "") {
  const [config, setConfig] = useState<OidcFederationConfig | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchConfig = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/oidc-federation-config`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data: OidcFederationConfig = await res.json();
      setConfig(data);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Unknown error");
    } finally {
      setLoading(false);
    }
  }, [baseUrl]);

  const updateConfig = useCallback(async (patch: Partial<OidcFederationConfig>) => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/oidc-federation-config`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(patch),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data: OidcFederationConfig = await res.json();
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
