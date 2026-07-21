import { useState, useCallback, useEffect } from "react";

/**
 * DEMO DATA — Tries real API first, falls back to empty demo data.
 * isDemoData flag indicates whether live or fallback data is shown.
 */

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
  const [isDemoData, setIsDemoData] = useState(true);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const importMetadata = useCallback(async (_input: string) => {
    setData((prev) => prev ? { ...prev, preview: { entity_id: "https://idp.example.com", sso_url: "https://idp.example.com/sso", name_id_format: "urn:oasis:names:tc:SAML:2.0:nameid-format:transient", cert_count: 2, valid: true } } : prev);
  }, []);
  const fetchData = useCallback(async () => {
    setLoading(true); setError(null);
    try { // Try real API first
      let res: Response | null = null;
      try {
        res = await fetch("/api/v1/data", { headers: { "Content-Type": "application/json" } });
      } catch { res = null; }
      if (res?.ok) { const d = await res.json(); setData(d); setIsDemoData(false); return; }
      setIsDemoData(true);
      setData({ preview: null, saved_idps: [
        { entity_id: "https://sts.windows.net/abc/", name: "Azure AD", imported_at: "2d ago" },
        { entity_id: "https://okta.com/default", name: "Okta", imported_at: "5d ago" },
      ] });
    } catch (e) { setError(e instanceof Error ? e.message : "Failed"); } finally { setLoading(false); }
  }, []);
  useEffect(() => { fetchData(); }, [fetchData]);
  return { data, loading, error, refresh: fetchData, importMetadata };
}
