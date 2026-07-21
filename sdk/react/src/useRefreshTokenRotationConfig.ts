import { useState, useCallback } from "react";

export interface RefreshTokenClientOverride {
  client_id: string;
  client_name: string;
  rotation_mode: "rotate" | "reuse";
  grace_period_seconds: number;
}

export interface RefreshTokenRotationConfig {
  rotation_mode: "rotate" | "reuse";
  reuse_detection: boolean;
  family_revocation_on_reuse: boolean;
  grace_period_seconds: number;
  backward_compat_duration: number;
  per_client_override: RefreshTokenClientOverride[];
}

export function useRefreshTokenRotationConfig(baseUrl: string = "") {
  const [config, setConfig] = useState<RefreshTokenRotationConfig | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchConfig = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/refresh-token-rotation-config`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setConfig(await res.json());
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); }
    finally { setLoading(false); }
  }, [baseUrl]);

  const updateConfig = useCallback(async (patch: Partial<RefreshTokenRotationConfig>) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/refresh-token-rotation-config`, {
        method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(patch),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json(); setConfig(data); return data;
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); return null; }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { config, loading, error, fetchConfig, updateConfig };
}
