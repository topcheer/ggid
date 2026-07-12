import { useState, useCallback } from "react";

export interface ContinuousMonitoringRule {
  control_id: string;
  description: string;
  frequency: "realtime" | "hourly" | "daily" | "weekly";
}

export interface FrameworkMapping {
  framework: string;
  controls_total: number;
  controls_met: number;
}

export interface RemediationTrigger {
  condition: string;
  action: string;
}

export interface ComplianceAutomationConfig {
  evidence_collection_schedule: string;
  continuous_monitoring_rules: ContinuousMonitoringRule[];
  framework_mapping: FrameworkMapping[];
  drift_detection: boolean;
  remediation_triggers: RemediationTrigger[];
  audit_readiness_score: number;
}

export function useComplianceAutomationConfig(baseUrl: string = "") {
  const [config, setConfig] = useState<ComplianceAutomationConfig | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const fetchConfig = useCallback(async () => {
    setLoading(true); setError(null);
    try { const res = await fetch(`${baseUrl}/api/v1/settings/compliance-automation-config`); if (!res.ok) throw new Error(`HTTP ${res.status}`); setConfig(await res.json()); }
    catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); } finally { setLoading(false); }
  }, [baseUrl]);
  const updateConfig = useCallback(async (patch: Partial<ComplianceAutomationConfig>) => {
    setLoading(true); setError(null);
    try { const res = await fetch(`${baseUrl}/api/v1/settings/compliance-automation-config`, { method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(patch) }); if (!res.ok) throw new Error(`HTTP ${res.status}`); const data = await res.json(); setConfig(data); return data; }
    catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); return null; } finally { setLoading(false); }
  }, [baseUrl]);
  return { config, loading, error, fetchConfig, updateConfig };
}
