import { useState, useCallback } from "react";

export interface PipelineStage {
  stage: string;
  description: string;
  enabled: boolean;
  config: Record<string, string>;
}

export interface ErrorRetryPolicy {
  max_attempts: number;
  backoff_seconds: number;
  dead_letter_queue: boolean;
}

export interface UserProvisioningPipelineConfig {
  pipeline_stages: PipelineStage[];
  webhook_notifications: { enabled: boolean; url: string };
  error_retry_policy: ErrorRetryPolicy;
  throughput_target: number;
}

export function useUserProvisioningPipelineConfig(baseUrl: string = "") {
  const [config, setConfig] = useState<UserProvisioningPipelineConfig | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchConfig = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/user-provisioning-pipeline-config`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setConfig(await res.json());
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); }
    finally { setLoading(false); }
  }, [baseUrl]);

  const updateConfig = useCallback(async (patch: Partial<UserProvisioningPipelineConfig>) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/user-provisioning-pipeline-config`, {
        method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(patch),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json(); setConfig(data); return data;
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); return null; }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { config, loading, error, fetchConfig, updateConfig };
}
