import { useState, useCallback } from "react";

export interface ResidencyRegion {
  region: string;
  allowed: boolean;
  encryption_required: boolean;
}

export interface CrossBorderRule {
  from_region: string;
  to_region: string;
  allowed: boolean;
  condition: string;
}

export interface SovereigntyViolation {
  timestamp: string;
  region: string;
  description: string;
  severity: "low" | "medium" | "high";
}

export interface DataSovereigntyConfig {
  residency_regions: ResidencyRegion[];
  cross_border_transfer_rules: CrossBorderRule[];
  gdpr_article_45_compliant: boolean;
  gdpr_article_49_compliant: boolean;
  data_localization_status: string;
  sovereignty_violations: SovereigntyViolation[];
}

export function useDataSovereigntyConfig(baseUrl: string = "") {
  const [config, setConfig] = useState<DataSovereigntyConfig | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchConfig = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/data-sovereignty-config`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setConfig(await res.json());
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); }
    finally { setLoading(false); }
  }, [baseUrl]);

  const updateConfig = useCallback(async (patch: Partial<DataSovereigntyConfig>) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/data-sovereignty-config`, {
        method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(patch),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json(); setConfig(data); return data;
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); return null; }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { config, loading, error, fetchConfig, updateConfig };
}
