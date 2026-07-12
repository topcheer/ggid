import { useState, useCallback, useEffect } from "react";

export interface FlaggedAccount {
  user: string;
  score: number;
  signals: string[];
  action_taken: string;
}

export interface VelocityRule {
  rule: string;
  threshold: number;
  current_rate: number;
  triggered_count: number;
}

export interface BlockedEntities {
  ips: number;
  emails: number;
  devices: number;
  total: number;
}

export interface FraudDetectionData {
  flagged_accounts: FlaggedAccount[];
  velocity_rules: VelocityRule[];
  device_fingerprint_count: number;
  blocked_entities: BlockedEntities;
  score_distribution: number[];
}

export function useFraudDetection() {
  const [data, setData] = useState<FraudDetectionData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      await new Promise((r) => setTimeout(r, 400));
      setData({
        flagged_accounts: [
          { user: "suspicious@temp.com", score: 92, signals: ["new_device", "velocity_burst", "known_bad_ip"], action_taken: "blocked" },
          { user: "user.account@ legit.com", score: 67, signals: ["device_mismatch", "off_hours"], action_taken: "challenged" },
          { user: "new.signup@mail.com", score: 45, signals: ["velocity_warning"], action_taken: "flagged" },
        ],
        velocity_rules: [
          { rule: "Signups per IP / hour", threshold: 10, current_rate: 7, triggered_count: 3 },
          { rule: "Login attempts / min", threshold: 5, current_rate: 2, triggered_count: 12 },
          { rule: "Password resets / hour", threshold: 3, current_rate: 1, triggered_count: 5 },
          { rule: "Account creations / day", threshold: 50, current_rate: 34, triggered_count: 0 },
        ],
        device_fingerprint_count: 15420,
        blocked_entities: { ips: 1247, emails: 89, devices: 34, total: 1370 },
        score_distribution: [120, 80, 200, 150, 90, 45, 30, 15, 8, 3],
      });
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load data");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);
  return { data, loading, error, refresh: fetchData };
}
