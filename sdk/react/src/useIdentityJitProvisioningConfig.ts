import { useState, useCallback, useEffect } from "react";

export interface IdpConfig {
  idp_name: string;
  enabled: boolean;
}

export interface AttributeMappingEntry {
  idp_claim: string;
  local_attr: string;
  required: boolean;
}

export interface ProvisioningLogEntry {
  user: string;
  idp: string;
  action: string;
  timestamp: string;
}

export interface IdentityJitProvisioningConfigData {
  per_idp_config: IdpConfig[];
  attribute_mapping: AttributeMappingEntry[];
  default_role_on_create: string;
  default_group_assignments: string[];
  update_on_login: boolean;
  conflict_resolution: string;
  provisioning_log_24h: ProvisioningLogEntry[];
}

export function useIdentityJitProvisioningConfig() {
  const [data, setData] = useState<IdentityJitProvisioningConfigData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      await new Promise((r) => setTimeout(r, 400));
      setData({
        per_idp_config: [
          { idp_name: "Google Workspace", enabled: true },
          { idp_name: "Azure AD", enabled: true },
          { idp_name: "Okta", enabled: false },
          { idp_name: "SAML IdP (Internal)", enabled: true },
        ],
        attribute_mapping: [
          { idp_claim: "email", local_attr: "email", required: true },
          { idp_claim: "name", local_attr: "full_name", required: true },
          { idp_claim: "department", local_attr: "department", required: false },
          { idp_claim: "groups", local_attr: "groups", required: false },
          { idp_claim: "picture", local_attr: "avatar_url", required: false },
        ],
        default_role_on_create: "developer",
        default_group_assignments: ["all-staff", "engineering"],
        update_on_login: true,
        conflict_resolution: "create",
        provisioning_log_24h: [
          { user: "new.user1@ggid.dev", idp: "Google Workspace", action: "created", timestamp: "10m ago" },
          { user: "returning.user@ggid.dev", idp: "Azure AD", action: "updated", timestamp: "30m ago" },
          { user: "third.user@ggid.dev", idp: "SAML IdP (Internal)", action: "created", timestamp: "2h ago" },
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
