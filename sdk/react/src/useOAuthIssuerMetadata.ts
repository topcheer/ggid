import { useState, useCallback, useEffect } from "react";

/**
 * DEMO DATA — Tries real API first, falls back to empty demo data.
 * isDemoData flag indicates whether live or fallback data is shown.
 */

export interface OAuthIssuerMetadataData {
  issuer_url: string;
  well_known_path: string;
  supported_response_types: string[];
  supported_subject_types: string[];
  claim_types_supported: string[];
  request_parameter_supported: boolean;
  request_uri_parameter_supported: boolean;
  require_request_uri_registration: boolean;
  well_known_preview: Record<string, unknown>;
}

export function useOAuthIssuerMetadata() {
  const [data, setData] = useState<OAuthIssuerMetadataData | null>(null);
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
        issuer_url: "https://auth.ggid.dev",
        well_known_path: "/.well-known/openid-configuration",
        supported_response_types: ["code", "token", "id_token", "code token", "code id_token", "code token id_token"],
        supported_subject_types: ["public", "pairwise"],
        claim_types_supported: ["normal", "aggregated", "distributed"],
        request_parameter_supported: true,
        request_uri_parameter_supported: true,
        require_request_uri_registration: false,
        well_known_preview: {
          issuer: "https://auth.ggid.dev",
          authorization_endpoint: "https://auth.ggid.dev/oauth/authorize",
          token_endpoint: "https://auth.ggid.dev/oauth/token",
          userinfo_endpoint: "https://auth.ggid.dev/oauth/userinfo",
          jwks_uri: "https://auth.ggid.dev/.well-known/jwks.json",
          response_types_supported: ["code", "token", "id_token"],
          subject_types_supported: ["public", "pairwise"],
          scopes_supported: ["openid", "profile", "email", "offline_access"],
          token_endpoint_auth_methods_supported: ["client_secret_basic", "client_secret_post", "private_key_jwt", "tls_client_auth"],
        },
      });
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load data");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);
  return { data, loading, error, refresh: fetchData, isDemoData };
}
