import { useState, useCallback } from "react";

export interface IdentityRecoveryConfig {
  takeover_response_checklist: string[];
  mass_reset_template: string;
  session_invalidation_scope: "affected" | "tenant" | "global";
  forensics_auto_collect: boolean;
  user_communication_template: string;
  notification_channels: string[];
}

export function useIdentityRecoveryConfig(baseUrl: string = "") {
  const [config, setConfig] = useState<IdentityRecoveryConfig | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const fetchConfig = useCallback(async () => {
    setLoading(true); setError(null);
    try { const res = await fetch(`${baseUrl}/api/v1/settings/identity-recovery-config`); if (!res.ok) throw new Error(`HTTP ${res.status}`); setConfig(await res.json()); }
    catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); } finally { setLoading(false); }
  }, [baseUrl]);
  const updateConfig = useCallback(async (patch: Partial<IdentityRecoveryConfig>) => {
    setLoading(true); setError(null);
    try { const res = await fetch(`${baseUrl}/api/v1/settings/identity-recovery-config`, { method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(patch) }); if (!res.ok) throw new Error(`HTTP ${res.status}`); const data = await res.json(); setConfig(data); return data; }
    catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); return null; } finally { setLoading(false); }
  }, [baseUrl]);
  return { config, loading, error, fetchConfig, updateConfig };
}
