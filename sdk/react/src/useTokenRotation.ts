import { useState, useCallback } from "react";

export interface RotationConfig {
  client_id: string;
  client_name: string;
  enabled: boolean;
  interval_days: number;
  max_age_hours: number;
  notify_before_hours: number;
}

export function useTokenRotation(baseUrl: string = "") {
  const [config, setConfig] = useState<RotationConfig | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchConfig = useCallback(async (clientId: string) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/oauth/token-rotation?client_id=${encodeURIComponent(clientId)}`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setConfig(await res.json());
    } catch (e: any) { setError(e.message); setConfig(null); }
    finally { setLoading(false); }
  }, [baseUrl]);

  const saveConfig = useCallback(async (cfg: RotationConfig) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/oauth/token-rotation`, { method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(cfg) });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setConfig(cfg); return true;
    } catch (e: any) { setError(e.message); return false; }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { config, loading, error, fetchConfig, saveConfig };
}
