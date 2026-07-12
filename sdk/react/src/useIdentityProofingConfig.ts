import { useState, useCallback } from "react";

export interface VerificationMethod {
  method: string;
  enabled: boolean;
}

export interface RiskLevelConfig {
  level: string;
  required_factors: number;
  methods: string[];
}

export interface IdentityProofingConfig {
  verification_methods: VerificationMethod[];
  required_factors: number;
  confidence_threshold: number;
  per_risk_level: RiskLevelConfig[];
  verification_provider: string;
  completion_rate: { total: number; completed: number; failed: number };
}

export function useIdentityProofingConfig(baseUrl: string = "") {
  const [config, setConfig] = useState<IdentityProofingConfig | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchConfig = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/identity-proofing-config`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setConfig(await res.json());
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); }
    finally { setLoading(false); }
  }, [baseUrl]);

  const updateConfig = useCallback(async (patch: Partial<IdentityProofingConfig>) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/identity-proofing-config`, {
        method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(patch),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json(); setConfig(data); return data;
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); return null; }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { config, loading, error, fetchConfig, updateConfig };
}
