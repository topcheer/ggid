import { useState, useCallback, useEffect } from "react";

export interface ClientUris {
  client_id: string;
  client_name: string;
  allowed_uris: string[];
}

export interface ValidationError {
  client_name: string;
  invalid_uri: string;
  reason: string;
}

export interface OAuthRedirectURIValidationData {
  exact_match_enabled: boolean;
  https_only: boolean;
  localhost_allowlist: boolean;
  custom_scheme_allowlist: string[];
  per_client_uris: ClientUris[];
  validation_errors: ValidationError[];
}

export function useOAuthRedirectURIValidation() {
  const [data, setData] = useState<OAuthRedirectURIValidationData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      await new Promise((r) => setTimeout(r, 400));
      setData({
        exact_match_enabled: true,
        https_only: true,
        localhost_allowlist: true,
        custom_scheme_allowlist: ["myapp://", "com.example.app://"],
        per_client_uris: [
          { client_id: "c-web-001", client_name: "Web Dashboard", allowed_uris: ["https://app.example.com/callback", "https://app.example.com/auth"] },
          { client_id: "c-mobile-002", client_name: "Mobile App", allowed_uris: ["myapp://callback", "https://app.example.com/mobile/callback"] },
          { client_id: "c-api-003", client_name: "API Service", allowed_uris: ["https://api.example.com/oauth/callback"] },
        ],
        validation_errors: [
          { client_name: "Legacy SPA", invalid_uri: "http://insecure.example.com/callback", reason: "HTTP not allowed when HTTPS Only is enabled" },
          { client_name: "Test Client", invalid_uri: "https://*.example.com/callback", reason: "Wildcard not allowed when Exact Match is enabled" },
        ],
      });
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load data");
    } finally {
      setLoading(false);
    }
  }, []);

  const testUri = useCallback(async (uri: string) => {
    console.log("Testing URI:", uri);
  }, []);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  return { data, loading, error, refresh: fetchData, testUri };
}
