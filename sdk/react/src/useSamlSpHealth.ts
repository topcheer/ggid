import { useState, useCallback } from "react";

export interface SpHealth {
  sp_entity_id: string;
  metadata_url: string;
  metadata_valid: boolean;
  cert_expiry_days: number;
  cert_expires_at: string;
  response_test: "pass" | "fail" | "untested";
  acs_url: string;
  acs_status: "ok" | "error" | "unknown";
  slo_url: string;
  slo_status: "ok" | "error" | "unknown";
  idp_connected: boolean;
  last_sync: string;
  errors: { timestamp: string; message: string }[];
}

export function useSamlSpHealth(baseUrl: string = "") {
  const [data, setData] = useState<SpHealth | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchHealth = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/oauth/saml-sp-health`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setData(await res.json());
    } catch (e: any) { setError(e.message); }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { data, loading, error, fetchHealth };
}
