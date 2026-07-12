import { useState, useCallback } from "react";

export interface RoleNode {
  name: string;
  level: number;
  parent: string | null;
}

export interface RoleTemplate {
  name: string;
  permissions: string[];
  description: string;
}

export interface SodPair {
  role_a: string;
  role_b: string;
  description: string;
}

export interface RbacDesignConfig {
  role_hierarchy: RoleNode[];
  max_depth: number;
  inheritance_enabled: boolean;
  role_templates: RoleTemplate[];
  sod_pairs: SodPair[];
  delegation_max_depth: number;
  auto_inherit_from_parent: boolean;
}

export function useRbacDesignConfig(baseUrl: string = "") {
  const [config, setConfig] = useState<RbacDesignConfig | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchConfig = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/rbac-design-config`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setConfig(await res.json());
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); }
    finally { setLoading(false); }
  }, [baseUrl]);

  const updateConfig = useCallback(async (patch: Partial<RbacDesignConfig>) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/rbac-design-config`, {
        method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(patch),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json(); setConfig(data); return data;
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); return null; }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { config, loading, error, fetchConfig, updateConfig };
}
