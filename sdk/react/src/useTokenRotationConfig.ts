import { useState, useCallback } from "react";

export interface TokenRotationEntry {
  client_id: string;
  client_name: string;
  rotation_interval_days: number;
  max_age_days: number;
  notify_before_days: number;
  auto_rotate: boolean;
  last_rotated: string;
}

export interface UpcomingRotation {
  client_id: string;
  client_name: string;
  rotation_due: string;
  days_until: number;
}

export interface TokenRotationConfig {
  per_client: TokenRotationEntry[];
  grace_period_hours: number;
  upcoming_rotations: { client_id: string; client_name: string; rotation_due: string; days_until: number }[];
}

export function useTokenRotationConfig(baseUrl: string = "") {
  const [config, setConfig] = useState<TokenRotationConfig | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchConfig = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/token-rotation-config`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setConfig(await res.json());
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); }
    finally { setLoading(false); }
  }, [baseUrl]);

  const updateConfig = useCallback(async (patch: Partial<TokenRotationConfig>) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/token-rotation-config`, {
        method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(patch),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json(); setConfig(data); return data;
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); return null; }
    finally { setLoading(false); }
  }, [baseUrl]);

  const bulkUpdate = useCallback(async (intervalDays: number, autoRotate: boolean) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/token-rotation-config/bulk`, {
        method: "POST", headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ rotation_interval_days: intervalDays, auto_rotate: autoRotate }),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json(); setConfig(data); return data;
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); return null; }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { config, loading, error, fetchConfig, updateConfig, bulkUpdate };
}
