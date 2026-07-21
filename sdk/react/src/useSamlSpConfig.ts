import { useState, useCallback } from "react";

export interface AttributeConsumingService {
  index: number;
  service_name: string;
  requested_attributes: string[];
}

export interface SamlSpConfig {
  entity_id: string;
  acs_url: string;
  slo_url: string;
  metadata_url: string;
  signature_algorithm: "RSA-SHA256" | "RSA-SHA1" | "ECDSA-SHA256";
  want_signed: boolean;
  want_encrypted: boolean;
  name_id_format: "unspecified" | "emailAddress" | "persistent" | "transient";
  attribute_consuming_service: AttributeConsumingService[];
}

export function useSamlSpConfig(baseUrl: string = "") {
  const [config, setConfig] = useState<SamlSpConfig | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchConfig = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/saml-sp-config`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setConfig(await res.json());
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); }
    finally { setLoading(false); }
  }, [baseUrl]);

  const updateConfig = useCallback(async (patch: Partial<SamlSpConfig>) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/saml-sp-config`, {
        method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(patch),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json(); setConfig(data); return data;
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); return null; }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { config, loading, error, fetchConfig, updateConfig };
}
