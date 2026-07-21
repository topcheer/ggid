import { useState, useCallback, useEffect } from "react";

/**
 * DEMO DATA — Tries real API first, falls back to empty demo data.
 * isDemoData flag indicates whether live or fallback data is shown.
 */

export interface SupportedAlg {
  name: string;
  cose_id: number;
}

export interface PlatformConfig {
  platform: string;
  authenticator_type: string;
  attachment: string;
  discoverable_credentials: boolean;
}

export interface AuthWebauthnConfigData {
  rp_id: string;
  rp_name: string;
  origin: string;
  attestation_requirement: "none" | "indirect" | "direct";
  user_verification: "required" | "preferred" | "discouraged";
  supported_algs: SupportedAlg[];
  timeout_seconds: number;
  per_platform_config: PlatformConfig[];
}

export function useAuthWebauthnConfig() {
  const [data, setData] = useState<AuthWebauthnConfigData | null>(null);
  const [isDemoData, setIsDemoData] = useState(true);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      // Try real API first
      let res: Response | null = null;
      try {
        res = await fetch("/api/v1/data", {
          headers: { "Content-Type": "application/json" },
        });
      } catch { res = null; }
      
      if (res?.ok) {
        const realData = await res.json();
        setData(realData);
        setIsDemoData(false);
        return;
      }
      
      // Fallback: empty demo data (no dangerous flags)
      setIsDemoData(true);
      setData({
        rp_id: "auth.ggid.dev",
        rp_name: "GGID Identity Platform",
        origin: "https://auth.ggid.dev",
        attestation_requirement: "direct",
        user_verification: "required",
        supported_algs: [
          { name: "RS256", cose_id: -257 },
          { name: "ES256", cose_id: -7 },
          { name: "EdDSA", cose_id: -8 },
        ],
        timeout_seconds: 300,
        per_platform_config: [
          { platform: "web", authenticator_type: "cross_platform", attachment: "cross_platform", discoverable_credentials: true },
          { platform: "ios", authenticator_type: "platform", attachment: "platform", discoverable_credentials: true },
          { platform: "android", authenticator_type: "platform", attachment: "platform", discoverable_credentials: true },
        ],
      });
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load data");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  return { data, loading, error, refresh: fetchData, isDemoData };
}
