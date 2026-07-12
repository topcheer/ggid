import { useState, useCallback } from "react";

export interface LifecycleStage {
  stage: string;
  sla_hours: number;
}

export interface AutoApprovalRule {
  condition: string;
  target_role: string;
  max_duration_days: number;
}

export interface AccessRequestLifecycleConfig {
  stages: LifecycleStage[];
  auto_approval_rules: AutoApprovalRule[];
  max_duration_days: number;
  renewal_reminder_days: number;
}

export function useAccessRequestLifecycleConfig(baseUrl: string = "") {
  const [config, setConfig] = useState<AccessRequestLifecycleConfig | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchConfig = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/access-request-lifecycle-config`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setConfig(await res.json());
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); }
    finally { setLoading(false); }
  }, [baseUrl]);

  const updateConfig = useCallback(async (patch: Partial<AccessRequestLifecycleConfig>) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/access-request-lifecycle-config`, {
        method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(patch),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json(); setConfig(data); return data;
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); return null; }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { config, loading, error, fetchConfig, updateConfig };
}
