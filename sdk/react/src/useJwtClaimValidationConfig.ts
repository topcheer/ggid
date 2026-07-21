import { useState, useCallback } from "react";

export interface RequiredClaim {
  claim: string;
  enabled: boolean;
}

export interface CustomClaim {
  name: string;
  type: "string" | "number" | "boolean" | "array";
  required: boolean;
  validator: string;
}

export interface JwtClaimValidationConfig {
  required_claims: { claim: string; enabled: boolean }[];
  clock_skew_seconds: number;
  validation_order: string[];
  custom_claims: CustomClaim[];
  strict_mode: boolean;
}

export function useJwtClaimValidationConfig(baseUrl: string = "") {
  const [config, setConfig] = useState<JwtClaimValidationConfig | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchConfig = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/jwt-claim-validation-config`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setConfig(await res.json());
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); }
    finally { setLoading(false); }
  }, [baseUrl]);

  const updateConfig = useCallback(async (patch: Partial<JwtClaimValidationConfig>) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/settings/jwt-claim-validation-config`, {
        method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(patch),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json(); setConfig(data); return data;
    } catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); return null; }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { config, loading, error, fetchConfig, updateConfig };
}
