import { useState, useCallback, useEffect } from "react";

export interface DiscoveryMethod {
  method: string;
  description: string;
  enabled: boolean;
}

export interface DomainRule {
  domain: string;
  provider_name: string;
  priority: string;
}

export interface DiscoveryLogEntry {
  id: string;
  timestamp: string;
  email: string;
  provider: string;
  result: string;
}

export interface IdpDiscoveryConfigData {
  discovery_methods: DiscoveryMethod[];
  email_domain_rules: DomainRule[];
  fallback_policy: string;
  discovery_log: DiscoveryLogEntry[];
}

export function useIdpDiscoveryConfig() {
  const [data, setData] = useState<IdpDiscoveryConfigData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      await new Promise((r) => setTimeout(r, 400));
      setData({
        discovery_methods: [
          { method: "Email Domain Mapping", description: "Map user email domain to configured IdP", enabled: true },
          { method: "HDR Discovery", description: "HTTP Header-based discovery via webfinger", enabled: true },
          { method: "LDAP Lookup", description: "Query LDAP for user's associated IdP", enabled: false },
        ],
        email_domain_rules: [
          { domain: "corp.com", provider_name: "Azure AD (corp.com)", priority: "1" },
          { domain: "partner.io", provider_name: "Okta (partner.io)", priority: "2" },
          { domain: "contractor.net", provider_name: "Auth0", priority: "3" },
        ],
        fallback_policy: "Show default login form with all providers",
        discovery_log: [
          { id: "1", timestamp: "2m ago", email: "alice@corp.com", provider: "Azure AD", result: "found" },
          { id: "2", timestamp: "15m ago", email: "bob@external.com", provider: "Local", result: "fallback" },
        ],
      });
    } catch (e) { setError(e instanceof Error ? e.message : "Failed"); }
    finally { setLoading(false); }
  }, []);

  const testDiscovery = useCallback((email: string) => { console.log("Testing discovery for", email); }, []);

  useEffect(() => { fetchData(); }, [fetchData]);
  return { data, loading, error, refresh: fetchData, testDiscovery };
}
