import { useState, useCallback } from "react";

export interface Attribute {
  name: string;
  pii_classification: "public" | "internal" | "confidential" | "restricted";
  mask_rule: string;
  access_frequency: number;
  last_accessed_by: string;
  last_accessed_at: string;
  retention_days: number;
}

export function useAttributeGovernance(baseUrl: string = "") {
  const [attributes, setAttributes] = useState<Attribute[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchAttributes = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/identity/attribute-governance`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json(); setAttributes(data.attributes || data || []);
    } catch (e: any) { setError(e.message); }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { attributes, loading, error, fetchAttributes };
}
