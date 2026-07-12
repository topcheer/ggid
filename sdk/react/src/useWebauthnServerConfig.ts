import { useState, useCallback } from "react";

export interface RpEntity {
  id: string;
  name: string;
  origins: string[];
}

export interface WebauthnServerConfig {
  ceremony_timeout: number;
  attestation_trust_path: "none" | "indirect" | "direct" | "enterprise";
  rp_entity: RpEntity;
  credential_storage_policy: "database" | "memory" | "hybrid";
  counter_enforcement: "strict" | "report";
  uv_preferred: boolean;
  aaguid_allowlist: string[];
}

export function useWebauthnServerConfig(baseUrl: string = "") {
  const [config, setConfig] = useState<WebauthnServerConfig | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchConfig = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/webauthn-server-config`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setConfig(await res.json());
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); }
    finally { setLoading(false); }
  }, [baseUrl]);

  const updateConfig = useCallback(async (patch: Partial<WebauthnServerConfig>) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/webauthn-server-config`, {
        method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(patch),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json(); setConfig(data); return data;
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); return null; }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { config, loading, error, fetchConfig, updateConfig };
}
