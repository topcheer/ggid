import { useState, useCallback, useEffect } from "react";

export interface TrustedTenant {
  tenant_id: string;
  trust_direction: string;
  scopes_allowed: string[];
  expires_at: string;
}

export interface AppSharing {
  app_id: string;
  app_name: string;
  shared_with_tenant_ids: string[];
  enabled: boolean;
}

export interface AuditEntry {
  action: string;
  actor: string;
  timestamp: string;
}

export interface RevocationPolicy {
  type: string;
  grace_period_hours: number;
}

export interface OAuthCrossTenantData {
  trusted_tenants: TrustedTenant[];
  revocation_policy: RevocationPolicy;
  audit_trail: AuditEntry[];
  per_app_sharing: AppSharing[];
}

export function useOAuthCrossTenant() {
  const [data, setData] = useState<OAuthCrossTenantData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      await new Promise((r) => setTimeout(r, 400));
      setData({
        trusted_tenants: [
          { tenant_id: "tenant-acme-corp", trust_direction: "bidirectional", scopes_allowed: ["openid", "profile", "email"], expires_at: "2025-12-31" },
          { tenant_id: "tenant-partner-org", trust_direction: "inbound", scopes_allowed: ["openid", "profile"], expires_at: "2025-09-30" },
          { tenant_id: "tenant-vendor-co", trust_direction: "outbound", scopes_allowed: ["openid"], expires_at: "2025-06-30" },
        ],
        revocation_policy: { type: "grace_period", grace_period_hours: 24 },
        audit_trail: [
          { action: "trust_created", actor: "admin@ggid.dev", timestamp: "2d ago" },
          { action: "scope_modified", actor: "admin@ggid.dev", timestamp: "5d ago" },
          { action: "trust_revoked", actor: "security@ggid.dev", timestamp: "1w ago" },
        ],
        per_app_sharing: [
          { app_id: "app-dashboard", app_name: "Analytics Dashboard", shared_with_tenant_ids: ["tenant-acme-corp"], enabled: true },
          { app_id: "app-mobile", app_name: "Mobile App", shared_with_tenant_ids: [], enabled: false },
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
