import { useState, useCallback } from "react";

export interface CustomAttribute {
  name: string;
  type: "string" | "number" | "boolean" | "date" | "reference";
  multi_valued: boolean;
  required: boolean;
  validation_rule: string;
  privacy_classification: "public" | "internal" | "confidential" | "restricted";
}

export interface IdentityAttributeSchemaConfig {
  standard_attributes: string[];
  custom_attributes: CustomAttribute[];
  schema_extension: boolean;
  per_attribute_masking: { attribute: string; masked: boolean }[];
}

export function useIdentityAttributeSchemaConfig(baseUrl: string = "") {
  const [config, setConfig] = useState<IdentityAttributeSchemaConfig | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchConfig = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/identity-attribute-schema-config`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setConfig(await res.json());
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); }
    finally { setLoading(false); }
  }, [baseUrl]);

  const updateConfig = useCallback(async (patch: Partial<IdentityAttributeSchemaConfig>) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/identity-attribute-schema-config`, {
        method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(patch),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json(); setConfig(data); return data;
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); return null; }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { config, loading, error, fetchConfig, updateConfig };
}
