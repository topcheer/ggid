import { useState, useCallback, useEffect } from "react";

/**
 * DEMO DATA — Tries real API first, falls back to empty demo data.
 * isDemoData flag indicates whether live or fallback data is shown.
 */

export interface ComplianceCheck {
  item: string;
  description: string;
  status: string;
}

export interface NonCompliantClient {
  client_id: string;
  issue: string;
  severity: string;
  remediation: string;
}

export interface RemediationAction {
  action: string;
  description: string;
  affected_count: number;
}

export interface OAuth21AuditData {
  compliance_checklist: ComplianceCheck[];
  non_compliant_clients: NonCompliantClient[];
  remediation_actions: RemediationAction[];
  overall_compliance_pct: number;
}

export function useOAuth21Audit() {
  const [data, setData] = useState<OAuth21AuditData | null>(null);
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
        compliance_checklist: [
          { item: "PKCE Required", description: "All authorization code flows must use PKCE (S256)", status: "pass" },
          { item: "Implicit Grant Disabled", description: "response_type=token must not be supported", status: "pass" },
          { item: "Password Grant Disabled", description: "grant_type=password must not be supported", status: "pass" },
          { item: "Exact Redirect URI Matching", description: "Redirect URIs must match exactly (no wildcards)", status: "pass" },
          { item: "State Parameter Mandatory", description: "state parameter required on all authorize requests", status: "pass" },
          { item: "DPoP Support", description: "Demonstration of Proof-of-Possession recommended", status: "warning" },
          { item: "Refresh Token Rotation", description: "Refresh token rotation should be enabled", status: "pass" },
          { item: "No http:// Redirect URIs", description: "Only https:// URIs allowed (except localhost)", status: "fail" },
        ],
        non_compliant_clients: [
          { client_id: "client-legacy-app", issue: "Uses http:// redirect URI", severity: "critical", remediation: "Switch to https:// or register localhost exception" },
          { client_id: "client-mobile-v1", issue: "No PKCE on code flow", severity: "high", remediation: "Implement PKCE with S256 challenge method" },
          { client_id: "client-spa-old", issue: "Uses implicit grant", severity: "high", remediation: "Migrate to authorization code + PKCE" },
        ],
        remediation_actions: [
          { action: "Enforce https:// for all redirect URIs", description: "Remove http:// exceptions except localhost", affected_count: 1 },
          { action: "Require PKCE for all public clients", description: "Block code flow requests without code_challenge", affected_count: 1 },
          { action: "Deprecate implicit grant", description: "Remove response_type=token support globally", affected_count: 1 },
          { action: "Enable DPoP", description: "Recommend DPoP for all SPA and mobile clients", affected_count: 8 },
        ],
        overall_compliance_pct: 87,
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
