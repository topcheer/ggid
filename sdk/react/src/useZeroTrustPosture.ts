import { useState, useCallback, useEffect } from "react";

export interface ZeroTrustViolation {
  type: string;
  severity: "low" | "medium" | "high" | "critical";
  timestamp: string;
}

export interface ZeroTrustPostureData {
  trust_score: number;
  device_compliance_pct: number;
  identity_verification_rate: number;
  network_segmentation: number;
  continuous_auth_coverage: number;
  violations_24h: number;
  recent_violations: ZeroTrustViolation[];
  posture_trend_30d: number[];
}

export function useZeroTrustPosture() {
  const [data, setData] = useState<ZeroTrustPostureData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      await new Promise((r) => setTimeout(r, 400));
      setData({
        trust_score: 78,
        device_compliance_pct: 92,
        identity_verification_rate: 96,
        network_segmentation: 85,
        continuous_auth_coverage: 71,
        violations_24h: 3,
        recent_violations: [
          { type: "Unmanaged device access attempt", severity: "high", timestamp: "2h ago" },
          { type: "MFA bypass detected", severity: "critical", timestamp: "5h ago" },
          { type: "Segmentation policy violation", severity: "medium", timestamp: "12h ago" },
        ],
        posture_trend_30d: Array.from({ length: 30 }, (_, i) => 70 + Math.round(Math.sin(i / 3) * 8 + i * 0.3)),
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

  return { data, loading, error, refresh: fetchData };
}
