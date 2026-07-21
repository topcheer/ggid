import { useState, useCallback } from "react";

export interface BruteForceConfig {
  max_attempts: number;
  lockout_duration_minutes: number;
  progressive_delay: boolean;
  captcha_threshold: number;
  ip_allowlist: string[];
  endpoint_overrides: { endpoint: string; max_attempts: number; lockout_minutes: number }[];
}

export function useBruteForceConfig(baseUrl: string = "") {
  const [config, setConfig] = useState<BruteForceConfig | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchConfig = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(baseUrl + "/api/v1/auth/brute-force-config");
      if (!res.ok) throw new Error("HTTP " + res.status);
      setConfig(await res.json());
    } catch (e: any) { setError(e.message); }
    finally { setLoading(false); }
  }, [baseUrl]);

  const saveConfig = useCallback(async (cfg: BruteForceConfig) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(baseUrl + "/api/v1/auth/brute-force-config", { method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(cfg) });
      if (!res.ok) throw new Error("HTTP " + res.status);
      setConfig(cfg); return true;
    } catch (e: any) { setError(e.message); return false; }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { config, loading, error, fetchConfig, saveConfig };
}
