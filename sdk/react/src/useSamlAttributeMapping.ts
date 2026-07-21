import { useState, useCallback } from "react";
export interface Mapping { id: string; source_attribute: string; target_field: string; transform: string; transform_value: string; }
export function useSamlAttributeMapping(baseUrl: string = "") {
  const [mappings, setMappings] = useState<Mapping[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const fetchMappings = useCallback(async () => { setLoading(true); setError(null); try { const res = await fetch(baseUrl + "/api/v1/auth/saml-attribute-mapping"); if (!res.ok) throw new Error("HTTP " + res.status); const d = await res.json(); setMappings(d.mappings || []); } catch (e: any) { setError(e.message); } finally { setLoading(false); } }, [baseUrl]);
  const addMapping = useCallback(async (m: Omit<Mapping, "id">) => { setLoading(true); setError(null); try { const res = await fetch(baseUrl + "/api/v1/auth/saml-attribute-mapping", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(m) }); if (!res.ok) throw new Error("HTTP " + res.status); return true; } catch (e: any) { setError(e.message); return false; } finally { setLoading(false); } }, [baseUrl]);
  return { mappings, loading, error, fetchMappings, addMapping };
}
