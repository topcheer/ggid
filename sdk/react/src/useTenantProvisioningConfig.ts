import { useState, useCallback } from "react";

export interface ProvisioningStep {
  step: string;
  description: string;
  enabled: boolean;
}

export interface OnboardingChecklistItem {
  item: string;
  completed: boolean;
}

export interface TenantProvisioningConfig {
  default_quota_template: string;
  provisioning_steps: ProvisioningStep[];
  auto_approve_new_tenants: boolean;
  trial_period_days: number;
  onboarding_checklist: OnboardingChecklistItem[];
}

export function useTenantProvisioningConfig(baseUrl: string = "") {
  const [config, setConfig] = useState<TenantProvisioningConfig | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchConfig = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/tenant-provisioning-config`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setConfig(await res.json());
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); }
    finally { setLoading(false); }
  }, [baseUrl]);

  const updateConfig = useCallback(async (patch: Partial<TenantProvisioningConfig>) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/tenant-provisioning-config`, {
        method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(patch),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json(); setConfig(data); return data;
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); return null; }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { config, loading, error, fetchConfig, updateConfig };
}
