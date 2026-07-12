import { useState, useCallback, useEffect } from "react";

export interface AppInfo {
  name: string;
  description: string;
}

export interface GrantTypeOption {
  value: string;
  description: string;
  selected: boolean;
}

export interface ScopeOption {
  name: string;
  description: string;
  required: boolean;
}

export interface GeneratedCredentials {
  client_id: string;
  client_secret: string;
}

export interface OAuthClientOnboardingWizardData {
  app_info: AppInfo;
  grant_types: GrantTypeOption[];
  redirect_uris: string[];
  scopes: ScopeOption[];
  credentials: GeneratedCredentials;
}

export function useOAuthClientOnboardingWizard() {
  const [data, setData] = useState<OAuthClientOnboardingWizardData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      await new Promise((r) => setTimeout(r, 400));
      setData({
        app_info: { name: "My OAuth App", description: "Sample application" },
        grant_types: [
          { value: "authorization_code", description: "For web/mobile apps with user consent", selected: true },
          { value: "client_credentials", description: "For machine-to-machine communication", selected: false },
          { value: "refresh_token", description: "Obtain new tokens after expiry", selected: true },
          { value: "urn:ietf:params:oauth:grant-type:device_code", description: "For devices without browsers", selected: false },
        ],
        redirect_uris: ["https://my-app.com/callback", "https://my-app.com/auth/silent"],
        scopes: [
          { name: "openid", description: "OpenID Connect authentication", required: true },
          { name: "profile", description: "Access to user profile claims", required: false },
          { name: "email", description: "Access to user email claim", required: false },
          { name: "offline_access", description: "Request refresh tokens", required: false },
        ],
        credentials: { client_id: "client_XyZ789abc123", client_secret: "secret_AbCdEfGhIjKlMnOpQrSt" },
      });
    } catch (e) { setError(e instanceof Error ? e.message : "Failed"); }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);
  return { data, loading, error, refresh: fetchData };
}
