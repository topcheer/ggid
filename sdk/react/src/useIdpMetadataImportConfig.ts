import { useState, useCallback, useEffect } from "react";

export interface MetadataPreview {
  entity_id: string;
  sso_url: string;
  name_id_format: string;
  cert_count: number;
  valid: boolean;
}

export interface SavedIdp {
  entity_id: string;
  name: string;
  imported_at: string;
}

export interface IdpMetadataImportConfigData {
  preview: MetadataPreview | null;
  saved_idps: SavedIdp[];
}

export function useIdpMetadataImportConfig() {
  const [data, setData] = useState<IdpMetadataImportConfigData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const importMetadata = useCallback(async (_input: string) => {
    setData((prev) => prev ? { ...prev, preview: { entity_id: "https://idp.example.com", sso_url: "https://idp.example.com/sso", name_id_format: "urn:oasis:names:tc:SAML:2.0:nameid-format:transient", cert_count: 2, valid: true } } : prev);
  }, []);
  const fetchData = useCallback(async () => {
    setLoading(true); setError(null);
    try { await new Promise((r) => setTimeout(r, 400));
      setData({ preview: null, saved_idps: [
        { entity_id: "https://sts.windows.net/abc/", name: "Azure AD", imported_at: "2d ago" },
        { entity_id: "https://okta.com/default", name: "Okta", imported_at: "5d ago" },
      ] });
    } catch (e) { setError(e instanceof Error ? e.message : "Failed"); } finally { setLoading(false); }
  }, []);
  useEffect(() => { fetchData(); }, [fetchData]);
  return { data, loading, error, refresh: fetchData, importMetadata };
}
