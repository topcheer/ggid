import { useState, useCallback } from "react";
export interface MetadataInfo { entity_id: string; sso_url: string; slo_url: string; name_id_format: string; certificates: string[]; }
export function useIdpMetadataImport(baseUrl: string = "") {
  const [metadata, setMetadata] = useState<MetadataInfo | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const preview = useCallback(async (source: string, type: "url" | "xml") => { setLoading(true); setError(null); try { const body = type === "url" ? { url: source } : { xml: source }; const res = await fetch(baseUrl + "/api/v1/auth/idp-metadata/preview", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) }); if (!res.ok) throw new Error("HTTP " + res.status); setMetadata(await res.json()); } catch (e: any) { setError(e.message); } finally { setLoading(false); } }, [baseUrl]);
  const importMeta = useCallback(async (source: string, type: "url" | "xml") => { setLoading(true); setError(null); try { const body = type === "url" ? { url: source } : { xml: source }; const res = await fetch(baseUrl + "/api/v1/auth/idp-metadata/import", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) }); if (!res.ok) throw new Error("HTTP " + res.status); return true; } catch (e: any) { setError(e.message); return false; } finally { setLoading(false); } }, [baseUrl]);
  return { metadata, loading, error, preview, importMeta };
}
