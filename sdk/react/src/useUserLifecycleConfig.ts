import { useState, useCallback } from "react";

export interface DormantDetectionRule {
  metric: string;
  threshold_days: number;
  enabled: boolean;
}

export interface StageTransitionRule {
  from_stage: string;
  to_stage: string;
  condition: string;
  auto: boolean;
}

export interface PerRoleOverride {
  role: string;
  deactivate_after_days: number;
  notify_before_days: number;
}

export interface UserLifecycleConfig {
  auto_deactivate_after_days: number;
  dormant_detection_rules: DormantDetectionRule[];
  stage_transition_rules: StageTransitionRule[];
  notification_before_deactivate: { enabled: boolean; days_before: number; channels: string[] };
  per_role_override: PerRoleOverride[];
}

export function useUserLifecycleConfig(baseUrl: string = "") {
  const [config, setConfig] = useState<UserLifecycleConfig | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchConfig = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/user-lifecycle-config`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setConfig(await res.json());
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); }
    finally { setLoading(false); }
  }, [baseUrl]);

  const updateConfig = useCallback(async (patch: Partial<UserLifecycleConfig>) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/user-lifecycle-config`, {
        method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(patch),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json(); setConfig(data); return data;
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); return null; }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { config, loading, error, fetchConfig, updateConfig };
}
