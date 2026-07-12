import { useState, useCallback } from "react";

export interface ScimMappingRule {
  source_field: string;
  target_field: string;
  required: boolean;
}

export interface ProvisioningTrigger {
  create: boolean;
  update: boolean;
  deactivate: boolean;
}

export interface ScimProvisioningConfig {
  endpoint_url: string;
  mapping_rules: ScimMappingRule[];
  provisioning_triggers: ProvisioningTrigger;
  sync_direction: "inbound" | "outbound" | "bidirectional";
  deprovision_on_disable: boolean;
  test_connection_status: "connected" | "disconnected" | "unknown";
}

export function useScimProvisioningConfig(baseUrl: string = "") {
  const [config, setConfig] = useState<ScimProvisioningConfig | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchConfig = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/scim-provisioning-config`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setConfig(await res.json());
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); }
    finally { setLoading(false); }
  }, [baseUrl]);

  const updateConfig = useCallback(async (patch: Partial<ScimProvisioningConfig>) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/scim-provisioning-config`, {
        method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(patch),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json(); setConfig(data); return data;
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); return null; }
    finally { setLoading(false); }
  }, [baseUrl]);

  const testConnection = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/scim-provisioning-config/test`, { method: "POST" });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      return await res.json();
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); return null; }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { config, loading, error, fetchConfig, updateConfig, testConnection };
}
