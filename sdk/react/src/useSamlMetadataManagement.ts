import { useState, useCallback } from "react";

export interface IdpMetadata {
  entity_id: string;
  url: string;
  last_refresh: string;
  signature_valid: boolean;
}

export interface SamlMetadataManagement {
  sp_metadata_preview: string;
  idp_metadata_list: IdpMetadata[];
  refresh_schedule_cron: string;
  federation_aggregation: boolean;
  entity_categories: string[];
}

export function useSamlMetadataManagement(baseUrl: string = "") {
  const [config, setConfig] = useState<SamlMetadataManagement | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchConfig = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/saml-metadata-management`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setConfig(await res.json());
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); }
    finally { setLoading(false); }
  }, [baseUrl]);

  const updateConfig = useCallback(async (patch: Partial<SamlMetadataManagement>) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/saml-metadata-management`, {
        method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(patch),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json(); setConfig(data); return data;
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); return null; }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { config, loading, error, fetchConfig, updateConfig };
}
