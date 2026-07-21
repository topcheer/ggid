import { useState, useCallback } from "react";

export interface PerAppBinding {
  application_id: string;
  application_name: string;
  binding_method: "cookie" | "bearer" | "DPoP" | "mTLS";
  rotation_interval: number;
}

export interface SessionBindingConfig {
  binding_method: "cookie" | "bearer" | "DPoP" | "mTLS";
  per_application_binding: PerAppBinding[];
  binding_rotation_policy: { interval_seconds: number; rotate_on_reauth: boolean };
  session_hijack_protection: boolean;
  fallback_method: "cookie" | "bearer";
  cross_device_transfer: { enabled: boolean; max_transfer_window: number };
}

export function useSessionBindingConfig(baseUrl: string = "") {
  const [config, setConfig] = useState<SessionBindingConfig | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchConfig = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/session-binding-config`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data: SessionBindingConfig = await res.json();
      setConfig(data);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Unknown error");
    } finally {
      setLoading(false);
    }
  }, [baseUrl]);

  const updateConfig = useCallback(async (patch: Partial<SessionBindingConfig>) => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/session-binding-config`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(patch),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data: SessionBindingConfig = await res.json();
      setConfig(data);
      return data;
    } catch (e) {
      setError(e instanceof Error ? e.message : "Unknown error");
      return null;
    } finally {
      setLoading(false);
    }
  }, [baseUrl]);

  return { config, loading, error, fetchConfig, updateConfig };
}
