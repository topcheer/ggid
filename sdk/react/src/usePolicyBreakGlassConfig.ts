import { useState, useCallback } from "react";

export interface BreakGlassRole {
  role: string;
  justification_required: boolean;
  auto_expire_minutes: number;
  notify_on_use: boolean;
}

export interface PolicyBreakGlassConfig {
  break_glass_roles: BreakGlassRole[];
  cooldown_period_minutes: number;
  max_concurrent: number;
  auto_revert: boolean;
}

export function usePolicyBreakGlassConfig(baseUrl: string = "") {
  const [config, setConfig] = useState<PolicyBreakGlassConfig | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchConfig = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/policy-break-glass-config`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setConfig(await res.json());
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); }
    finally { setLoading(false); }
  }, [baseUrl]);

  const updateConfig = useCallback(async (patch: Partial<PolicyBreakGlassConfig>) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/policy-break-glass-config`, {
        method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(patch),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json(); setConfig(data); return data;
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); return null; }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { config, loading, error, fetchConfig, updateConfig };
}
