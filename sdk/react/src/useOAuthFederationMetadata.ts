import { useState, useCallback, useEffect } from "react";

/**
 * DEMO DATA — Tries real API first, falls back to empty demo data.
 * isDemoData flag indicates whether live or fallback data is shown.
 */

export interface FederatedEntity {
  entity_id: string;
  role: string;
  metadata_url: string;
  trust_status: string;
  last_refresh: string;
  entity_categories: string[];
}

export interface OAuthFederationMetadataData {
  federated_entities: FederatedEntity[];
  auto_refresh_schedule: string;
  next_refresh: string;
}

export function useOAuthFederationMetadata() {
  const [data, setData] = useState<OAuthFederationMetadataData | null>(null);
  const [isDemoData, setIsDemoData] = useState(true);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      // Try real API first
      let res: Response | null = null;
      try {
        res = await fetch("/api/v1/data", { headers: { "Content-Type": "application/json" } });
      } catch { res = null; }
      if (res?.ok) { const d = await res.json(); setData(d); setIsDemoData(false); return; }
      setIsDemoData(true);
      setData({
        federated_entities: [
          { entity_id: "https://idp.corp.com", role: "OP (OpenID Provider)", metadata_url: "https://idp.corp.com/.well-known/openid-configuration", trust_status: "trusted", last_refresh: "2h ago", entity_categories: ["SPID", "REFEDS"] },
          { entity_id: "https://rp.partner.io", role: "RP (Relying Party)", metadata_url: "https://rp.partner.io/metadata.xml", trust_status: "trusted", last_refresh: "5h ago", entity_categories: ["REFEDS"] },
          { entity_id: "https://auth.contractor.net", role: "OP", metadata_url: "https://auth.contractor.net/.well-known/openid-configuration", trust_status: "pending", last_refresh: "1d ago", entity_categories: [] },
        ],
        auto_refresh_schedule: "Every 24h (02:00 UTC)",
        next_refresh: "in 18h",
      });
    } catch (e) { setError(e instanceof Error ? e.message : "Failed"); }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);
  return { data, loading, error, refresh: fetchData, isDemoData };
}
