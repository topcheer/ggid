import { useState, useCallback } from "react";

export interface ReEnrollmentStep {
  step: string;
  description: string;
  required: boolean;
}

export interface WebauthnRecoveryConfig {
  backup_authenticator_required: boolean;
  max_devices_per_user: number;
  recovery_codes_count: number;
  recovery_code_format: "numeric" | "alphanumeric" | "hex";
  re_enrollment_flow: ReEnrollmentStep[];
  admin_assisted_recovery: boolean;
}

export function useWebauthnRecoveryConfig(baseUrl: string = "") {
  const [config, setConfig] = useState<WebauthnRecoveryConfig | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchConfig = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/webauthn-recovery-config`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setConfig(await res.json());
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); }
    finally { setLoading(false); }
  }, [baseUrl]);

  const updateConfig = useCallback(async (patch: Partial<WebauthnRecoveryConfig>) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/webauthn-recovery-config`, {
        method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(patch),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json(); setConfig(data); return data;
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); return null; }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { config, loading, error, fetchConfig, updateConfig };
}
