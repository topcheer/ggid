import { useState, useCallback } from "react";

export interface AttributeSource {
  category: "user" | "resource" | "env" | "request";
  attributes: string[];
}

export interface ConditionTemplate {
  id: string;
  name: string;
  expression: string;
  description: string;
}

export interface AbacConditionConfig {
  attribute_sources: AttributeSource[];
  operators_per_type: Record<string, string[]>;
  condition_templates: ConditionTemplate[];
  evaluation_cache_ttl: number;
  default_deny: boolean;
}

export function useAbacConditionConfig(baseUrl: string = "") {
  const [config, setConfig] = useState<AbacConditionConfig | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchConfig = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/abac-condition-config`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setConfig(await res.json());
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); }
    finally { setLoading(false); }
  }, [baseUrl]);

  const updateConfig = useCallback(async (patch: Partial<AbacConditionConfig>) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/abac-condition-config`, {
        method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(patch),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json(); setConfig(data); return data;
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); return null; }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { config, loading, error, fetchConfig, updateConfig };
}
