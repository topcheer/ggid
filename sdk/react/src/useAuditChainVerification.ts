import { useState, useCallback, useEffect } from "react";

export interface LastVerification {
  blocks_verified: number;
  blocks_failed: number;
  chain_integrity_pct: number;
}

export interface VerificationLogEntry {
  run_id: string;
  timestamp: string;
  result: string;
  duration_ms: number;
  anomalies_found: number;
}

export interface AuditChainVerificationData {
  last_verification: LastVerification;
  verification_log: VerificationLogEntry[];
  auto_verify_schedule: string;
  alert_on_failure: boolean;
}

export function useAuditChainVerification() {
  const [data, setData] = useState<AuditChainVerificationData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      await new Promise((r) => setTimeout(r, 400));
      setData({
        last_verification: { blocks_verified: 1284500, blocks_failed: 0, chain_integrity_pct: 100 },
        verification_log: [
          { run_id: "ver-005", timestamp: "5m ago", result: "pass", duration_ms: 3400, anomalies_found: 0 },
          { run_id: "ver-004", timestamp: "1h ago", result: "pass", duration_ms: 3200, anomalies_found: 0 },
          { run_id: "ver-003", timestamp: "2h ago", result: "pass", duration_ms: 3500, anomalies_found: 0 },
          { run_id: "ver-002", timestamp: "3h ago", result: "warning", duration_ms: 4100, anomalies_found: 2 },
          { run_id: "ver-001", timestamp: "4h ago", result: "pass", duration_ms: 3100, anomalies_found: 0 },
        ],
        auto_verify_schedule: "0 */1 * * *",
        alert_on_failure: true,
      });
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load data");
    } finally {
      setLoading(false);
    }
  }, []);

  const verifyNow = useCallback(async () => {
    console.log("Verifying chain now");
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);
  return { data, loading, error, refresh: fetchData, verifyNow };
}
