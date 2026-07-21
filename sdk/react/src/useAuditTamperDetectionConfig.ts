import { useState, useCallback } from "react";

export interface AuditTamperDetectionConfig {
  verify_interval_minutes: number;
  alert_on_tamper: boolean;
  insertion_gap_threshold: number;
  reorder_detection_sensitivity: "low" | "medium" | "high";
  forensics_auto_collection: boolean;
  recovery_procedure_template: string;
}

export function useAuditTamperDetectionConfig(baseUrl: string = "") {
  const [config, setConfig] = useState<AuditTamperDetectionConfig | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchConfig = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/audit-tamper-detection-config`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setConfig(await res.json());
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); }
    finally { setLoading(false); }
  }, [baseUrl]);

  const updateConfig = useCallback(async (patch: Partial<AuditTamperDetectionConfig>) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/audit-tamper-detection-config`, {
        method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(patch),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json(); setConfig(data); return data;
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); return null; }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { config, loading, error, fetchConfig, updateConfig };
}
