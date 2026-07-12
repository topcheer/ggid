import { useState, useCallback, useEffect } from "react";

export interface TrustAnchor {
  issuer: string;
  jwks_uri: string;
  trust_mark_valid: boolean;
}

export interface FederatedProvider {
  entity_id: string;
  organization: string;
  role: string;
  status: "active" | "pending" | "suspended";
}

export interface TrustChainNode {
  entity: string;
  metadata_type: string;
  verified: boolean;
}

export interface AuthOidcFederationData {
  trust_anchors: TrustAnchor[];
  federated_providers: FederatedProvider[];
  trust_chain: TrustChainNode[];
  entity_statement: Record<string, unknown>;
  trust_resolution_status: "healthy" | "degraded" | "error";
  last_auto_discovery: string;
}

export function useAuthOidcFederation() {
  const [data, setData] = useState<AuthOidcFederationData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      await new Promise((r) => setTimeout(r, 400));
      setData({
        trust_anchors: [
          { issuer: "https://fed.example.org", jwks_uri: "https://fed.example.org/.well-known/jwks.json", trust_mark_valid: true },
          { issuer: "https://swamid.se", jwks_uri: "https://swamid.se/.well-known/jwks.json", trust_mark_valid: true },
          { issuer: "https://incommon.org", jwks_uri: "https://incommon.org/.well-known/jwks.json", trust_mark_valid: false },
        ],
        federated_providers: [
          { entity_id: "https://idp.univ1.edu", organization: "University One", role: "OP", status: "active" },
          { entity_id: "https://idp.univ2.edu", organization: "University Two", role: "OP", status: "active" },
          { entity_id: "https://idp.univ3.edu", organization: "University Three", role: "OP", status: "pending" },
          { entity_id: "https://sp.research.org", organization: "Research Portal", role: "RP", status: "active" },
        ],
        trust_chain: [
          { entity: "Trust Anchor (fed.example.org)", metadata_type: "federation_entity", verified: true },
          { entity: "Intermediate (edu-federation)", metadata_type: "federation_entity", verified: true },
          { entity: "Identity Provider (idp.univ1.edu)", metadata_type: "openid_provider", verified: true },
          { entity: "Relying Party (sp.research.org)", metadata_type: "openid_relying_party", verified: false },
        ],
        entity_statement: {
          iss: "https://fed.example.org",
          sub: "https://idp.univ1.edu",
          iat: 1700000000,
          exp: 1700086400,
          metadata: {
            openid_provider: {
              issuer: "https://idp.univ1.edu",
              authorization_endpoint: "https://idp.univ1.edu/authorize",
              token_endpoint: "https://idp.univ1.edu/token",
              jwks_uri: "https://idp.univ1.edu/jwks",
            },
          },
          constraints: { max_path_length: 2 },
        },
        trust_resolution_status: "healthy",
        last_auto_discovery: "5m ago",
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
