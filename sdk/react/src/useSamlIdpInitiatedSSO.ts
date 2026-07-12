import { useState, useCallback, useEffect } from "react";

export interface AllowedIdp {
  entity_id: string;
  provider_name: string;
  status: "active" | "pending" | "disabled";
  idp_initiated_enabled: boolean;
}

export interface SessionBridge {
  create_local_session: boolean;
  map_attributes: string[];
}

export interface SamlIdpInitiatedSSOData {
  allowed_idps: AllowedIdp[];
  relay_state_config: string;
  sso_url_preview: string;
  session_bridge: SessionBridge;
  security_warnings: string[];
}

export function useSamlIdpInitiatedSSO() {
  const [data, setData] = useState<SamlIdpInitiatedSSOData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      await new Promise((r) => setTimeout(r, 400));
      setData({
        allowed_idps: [
          { entity_id: "https://idp.univ.edu/idp", provider_name: "University IdP", status: "active", idp_initiated_enabled: true },
          { entity_id: "https://corp.okta.com", provider_name: "Okta Corporate", status: "active", idp_initiated_enabled: true },
          { entity_id: "https://ad.corp.local", provider_name: "Active Directory FS", status: "pending", idp_initiated_enabled: false },
        ],
        relay_state_config: "sp:app1.home",
        sso_url_preview: "https://idp.example.com/idp/profile/SAML2/Unsolicited/SSO?providerId=https://sp.example.com",
        session_bridge: {
          create_local_session: true,
          map_attributes: ["email", "displayname", "groups", "uid"],
        },
        security_warnings: [
          "IdP-initiated SSO can be exploited if RelayState is not validated.",
          "Ensure all IdP entity IDs are explicitly allowlisted.",
          "Consider requiring reauthentication for sensitive applications.",
        ],
      });
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load data");
    } finally {
      setLoading(false);
    }
  }, []);

  const testSso = useCallback(async () => {
    console.log("Testing IdP-initiated SSO");
  }, []);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  return { data, loading, error, refresh: fetchData, testSso };
}
