import { useState, useCallback, useEffect } from "react";

export interface VelocityRule {
  rule_name: string;
  metric: string;
  window: string;
  threshold: number;
  action: string;
  current_rate: number;
  triggered_24h: number;
}

export interface VelocityRulesConfigData {
  rules: VelocityRule[];
  scope: string;
  geographic_velocity_check: boolean;
}

export function useVelocityRulesConfig() {
  const [data, setData] = useState<VelocityRulesConfigData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      await new Promise((r) => setTimeout(r, 400));
      setData({
        rules: [
          { rule_name: "Rapid Signups", metric: "registrations", window: "per_hour", threshold: 10, action: "challenge", current_rate: 4, triggered_24h: 3 },
          { rule_name: "Login Brute Force", metric: "logins", window: "per_minute", threshold: 5, action: "block", current_rate: 2, triggered_24h: 12 },
          { rule_name: "Password Spray", metric: "password_changes", window: "per_hour", threshold: 20, action: "block", current_rate: 8, triggered_24h: 5 },
          { rule_name: "API Rate Limit", metric: "api_calls", window: "per_minute", threshold: 100, action: "throttle", current_rate: 67, triggered_24h: 28 },
          { rule_name: "Account Recovery Abuse", metric: "password_changes", window: "per_day", threshold: 3, action: "challenge", current_rate: 1, triggered_24h: 2 },
        ],
        scope: "per_device",
        geographic_velocity_check: true,
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
