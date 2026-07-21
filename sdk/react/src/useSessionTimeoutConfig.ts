import { useState, useCallback } from "react";

export interface SessionTimeoutConfig {
  idle_timeout_minutes: number;
  absolute_timeout_hours: number;
  warning_before_minutes: number;
  grace_period: boolean;
  enforce_on_mobile: boolean;
  role_overrides: { role: string; idle_timeout_minutes: number; absolute_timeout_hours: number }[];
}

export function useSessionTimeoutConfig(baseUrl: string = "") {
  const [config, setConfig] = useState<SessionTimeoutConfig | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchConfig = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(baseUrl + "/api/v1/auth/session-timeout-config");
      if (!res.ok) throw new Error("HTTP " + res.status);
      setConfig(await res.json());
    } catch (e: any) { setError(e.message); }
    finally { setLoading(false); }
  }, [baseUrl]);

  const saveConfig = useCallback(async (cfg: SessionTimeoutConfig) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(baseUrl + "/api/v1/auth/session-timeout-config", { method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(cfg) });
      if (!res.ok) throw new Error("HTTP " + res.status);
      setConfig(cfg); return true;
    } catch (e: any) { setError(e.message); return false; }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { config, loading, error, fetchConfig, saveConfig };
}
