import { useState, useCallback, useEffect } from "react";

/**
 * DEMO DATA — Tries real API first, falls back to empty demo data.
 */

export interface SupportedAlgEntry {
  id: string;
  cose_id: number;
  enabled: boolean;
}

export interface PlatformEntry {
  platform: string;
  authenticator_type: string;
  attachment: string;
  enabled: boolean;
}

export interface WebauthnConfigData {
  rp_id: string;
  rp_name: string;
  origin: string;
  attestation_requirement: string;
  user_verification: string;
  timeout_seconds: number;
  supported_algorithms: SupportedAlgEntry[];
  per_platform: PlatformEntry[];
}

export function useWebauthnConfig() {
  const [data, setData] = useState<WebauthnConfigData | null>(null);
  const [isDemoData, setIsDemoData] = useState(true);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const fetchData = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      // Try real API first
      let res: Response | null = null;
      try { res = await fetch("/api/v1/data", { headers: { "Content-Type": "application/json" } }); } catch { res = null; }
      if (res?.ok) { const d = await res.json(); setData(d); setIsDemoData(false); return; }
      setIsDemoData(true);
      setData({
        rp_id: "auth.ggid.dev", rp_name: "GGID", origin: "https://auth.ggid.dev",
        attestation_requirement: "indirect", user_verification: "preferred", timeout_seconds: 300,
        supported_algorithms: [
          { id: "RS256", cose_id: -257, enabled: true },
          { id: "ES256", cose_id: -7, enabled: true },
          { id: "EdDSA", cose_id: -8, enabled: false },
        ],
        per_platform: [
          { platform: "Web (Chrome/Edge)", authenticator_type: "platform", attachment: "internal", enabled: true },
          { platform: "iOS (Face ID/Touch ID)", authenticator_type: "platform", attachment: "internal", enabled: true },
          { platform: "Security Key (USB/NFC)", authenticator_type: "cross-platform", attachment: "cross-platform", enabled: true },
        ],
      });
    } catch (e) { setError(e instanceof Error ? e.message : "Failed"); }
    finally { setLoading(false); }
  }, []);
  useEffect(() => { fetchData(); }, [fetchData]);
  return { data, loading, error, refresh: fetchData, isDemoData };
}
