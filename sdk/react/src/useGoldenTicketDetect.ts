import { useState, useCallback, useEffect } from "react";

/**
 * DEMO DATA — Tries real API first, falls back to empty demo data.
 * isDemoData flag indicates whether live or fallback data is shown.
 */

export interface DetectedForgery {
  token_hash: string;
  anomaly_type: string;
  user: string;
  source_ip: string;
  timestamp: string;
}

export interface ForgeryDetectionRule {
  rule_name: string;
  description: string;
  enabled: boolean;
}

export interface GoldenTicketDetectData {
  detected_forgeries: DetectedForgery[];
  detection_rules: ForgeryDetectionRule[];
  false_positive_rate_pct: number;
  auto_revoke_enabled: boolean;
}

export function useGoldenTicketDetect() {
  const [data, setData] = useState<GoldenTicketDetectData | null>(null);
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
        detected_forgeries: [
          { token_hash: "a1b2c3d4e5f6789abcdef0123456789", anomaly_type: "issuer_mismatch", user: "admin@ggid.dev", source_ip: "10.0.5.23", timestamp: "15m ago" },
          { token_hash: "b2c3d4e5f67890abcdef12345678901", anomaly_type: "signature_anomaly", user: "svc.legacy", source_ip: "192.168.1.50", timestamp: "1h ago" },
          { token_hash: "c3d4e5f67890abcdef23456789012", anomaly_type: "abnormal_claims", user: "user.test", source_ip: "172.16.0.8", timestamp: "2h ago" },
          { token_hash: "d4e5f67890abcdef345678901234", anomaly_type: "expiry_anomaly", user: "admin.old", source_ip: "10.0.5.99", timestamp: "3h ago" },
        ],
        detection_rules: [
          { rule_name: "Issuer validation", description: "Check token issuer matches configured IdP", enabled: true },
          { rule_name: "Signature verification", description: "Validate token signature against JWKS", enabled: true },
          { rule_name: "Claim anomaly detection", description: "Detect unusual claim patterns (admin claims on service tokens)", enabled: true },
          { rule_name: "Expiry anomaly detection", description: "Flag tokens with abnormal expiry times (>7d)", enabled: true },
          { rule_name: "Token replay detection", description: "Check jti against known replay databases", enabled: false },
        ],
        false_positive_rate_pct: 3,
        auto_revoke_enabled: false,
      });
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load data");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);
  return { data, loading, error, refresh: fetchData, isDemoData };
}
