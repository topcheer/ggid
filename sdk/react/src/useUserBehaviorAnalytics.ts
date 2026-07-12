import { useState, useCallback, useEffect } from "react";

export interface BehavioralBaseline {
  login_time_range: string;
  typical_devices: string[];
  geo_patterns: string[];
  access_patterns: string[];
}

export interface BehaviorAnomaly {
  type: "unusual_time" | "new_device" | "new_location" | "unusual_access";
  description: string;
  severity: "low" | "medium" | "high";
  timestamp: string;
}

export interface UserBehaviorAnalyticsData {
  baseline: BehavioralBaseline;
  deviation_score: number;
  anomalies: BehaviorAnomaly[];
  trend_7d: number[];
}

export function useUserBehaviorAnalytics() {
  const [data, setData] = useState<UserBehaviorAnalyticsData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      await new Promise((r) => setTimeout(r, 400));
      setData({
        baseline: {
          login_time_range: "08:00 - 18:00 UTC",
          typical_devices: ["MacBook Pro (Work)", "iPhone 15"],
          geo_patterns: ["San Francisco, CA", "Remote: US West"],
          access_patterns: ["Dashboard read", "User list view", "Audit query"],
        },
        deviation_score: 35,
        anomalies: [
          { type: "unusual_time", description: "Login at 03:14 UTC (outside baseline 08:00-18:00)", severity: "medium", timestamp: "2h ago" },
          { type: "new_device", description: "Access from unrecognized Android device", severity: "high", timestamp: "5h ago" },
          { type: "new_location", description: "Login from Lagos, Nigeria (new geo)", severity: "high", timestamp: "1d ago" },
          { type: "unusual_access", description: "Bulk export of user data (outside baseline)", severity: "medium", timestamp: "2d ago" },
        ],
        trend_7d: [15, 22, 18, 45, 35, 50, 35],
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
