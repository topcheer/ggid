import { useState, useCallback } from "react";

export interface DeviceClientEntry {
  client_id: string;
  client_name: string;
  enabled: boolean;
}

export interface DeviceFlowStats {
  completed: number;
  expired: number;
  rejected: number;
}

export interface DeviceAuthorizationFlowConfig {
  device_code_lifetime: number;
  polling_interval: number;
  user_code_format: "numeric" | "alphanumeric";
  per_client_enabled: DeviceClientEntry[];
  qr_code_enabled: boolean;
  stats: DeviceFlowStats;
}

export function useDeviceAuthorizationFlowConfig(baseUrl: string = "") {
  const [config, setConfig] = useState<DeviceAuthorizationFlowConfig | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchConfig = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/device-authorization-flow-config`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setConfig(await res.json());
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); }
    finally { setLoading(false); }
  }, [baseUrl]);

  const updateConfig = useCallback(async (patch: Partial<DeviceAuthorizationFlowConfig>) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/device-authorization-flow-config`, {
        method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(patch),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json(); setConfig(data); return data;
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); return null; }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { config, loading, error, fetchConfig, updateConfig };
}
