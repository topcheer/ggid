import { useState, useCallback, useEffect } from "react";

export interface OAuthIssuerMetadataConfigData {
  issuer_url: string;
  response_types: string[];
  subject_types: string[];
  claim_types: string[];
  request_param_supported: boolean;
  request_uri_supported: boolean;
  require_request_uri: boolean;
  well_known: Record<string, unknown>;
}

export function useOAuthIssuerMetadataConfig() {
  const [data, setData] = useState<OAuthIssuerMetadataConfigData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const fetchData = useCallback(async () => {
    setLoading(true); setError(null);
    try { await new Promise((r) => setTimeout(r, 400));
      setData({
        issuer_url: "https://auth.ggid.dev",
        response_types: ["code", "token", "id_token", "code id_token", "code token"],
        subject_types: ["public", "pairwise"],
        claim_types: ["normal", "aggregated", "distributed"],
        request_param_supported: true, request_uri_supported: true, require_request_uri: false,
        well_known: { issuer: "https://auth.ggid.dev", authorization_endpoint: "https://auth.ggid.dev/oauth/authorize", token_endpoint: "https://auth.ggid.dev/oauth/token", jwks_uri: "https://auth.ggid.dev/.well-known/jwks.json", scopes_supported: ["openid", "profile", "email", "offline_access"], response_types_supported: ["code", "token", "id_token"] },
      });
    } catch (e) { setError(e instanceof Error ? e.message : "Failed"); } finally { setLoading(false); }
  }, []);
  useEffect(() => { fetchData(); }, [fetchData]);
  return { data, loading, error, refresh: fetchData };
}
