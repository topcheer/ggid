import { useState, useCallback } from "react";

export interface ComplianceCheckItem {
  item: string;
  status: "pass" | "fail" | "warn";
  detail: string;
}

export interface NonCompliantClient {
  client_id: string;
  client_name: string;
  issues: string[];
}

export interface RemediationAction {
  action: string;
  priority: "high" | "medium" | "low";
}

export interface OAuth21ComplianceChecker {
  checklist: ComplianceCheckItem[];
  non_compliant_clients: NonCompliantClient[];
  remediation_actions: RemediationAction[];
  overall_pct: number;
}

export function useOAuth21ComplianceChecker(baseUrl: string = "") {
  const [config, setConfig] = useState<OAuth21ComplianceChecker | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchConfig = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/oauth-2-1-compliance-checker`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setConfig(await res.json());
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); }
    finally { setLoading(false); }
  }, [baseUrl]);

  const updateConfig = useCallback(async (patch: Partial<OAuth21ComplianceChecker>) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/oauth-2-1-compliance-checker`, {
        method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(patch),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json(); setConfig(data); return data;
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); return null; }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { config, loading, error, fetchConfig, updateConfig };
}
