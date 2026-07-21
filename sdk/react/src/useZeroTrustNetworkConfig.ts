import { useState, useCallback } from "react";

export interface DeviceTrustSignal {
  signal: string;
  source: string;
  weight: number;
}

export interface NetworkAccessPolicy {
  resource: string;
  required_trust_level: "low" | "medium" | "high";
  condition: string;
}

export interface ZeroTrustNetworkConfig {
  identity_aware_proxy: boolean;
  continuous_verification_interval: number;
  device_trust_signals: DeviceTrustSignal[];
  network_access_policy: NetworkAccessPolicy[];
  microsegmentation_rules: string[];
}

export function useZeroTrustNetworkConfig(baseUrl: string = "") {
  const [config, setConfig] = useState<ZeroTrustNetworkConfig | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const fetchConfig = useCallback(async () => {
    setLoading(true); setError(null);
    try { const res = await fetch(`${baseUrl}/api/v1/settings/zero-trust-network-config`); if (!res.ok) throw new Error(`HTTP ${res.status}`); setConfig(await res.json()); }
    catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); } finally { setLoading(false); }
  }, [baseUrl]);
  const updateConfig = useCallback(async (patch: Partial<ZeroTrustNetworkConfig>) => {
    setLoading(true); setError(null);
    try { const res = await fetch(`${baseUrl}/api/v1/settings/zero-trust-network-config`, { method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(patch) }); if (!res.ok) throw new Error(`HTTP ${res.status}`); const data = await res.json(); setConfig(data); return data; }
    catch (e) { setError(e instanceof Error ? e.message : "Unknown error"); return null; } finally { setLoading(false); }
  }, [baseUrl]);
  return { config, loading, error, fetchConfig, updateConfig };
}
