import { useState, useCallback, useEffect } from "react";

export interface TimeWindowRule {
  policy: string;
  allowed_days: string[];
  start_time: string;
  end_time: string;
  timezone: string;
}

export interface RoleRestriction {
  role: string;
  allowed_days: string[];
  start_time: string;
  end_time: string;
}

export interface Holiday {
  name: string;
  date: string;
  access: string;
}

export interface PolicyTimeBasedAccessData {
  time_window_rules: TimeWindowRule[];
  per_role_restrictions: RoleRestriction[];
  holiday_calendar: Holiday[];
  grace_period_minutes: number;
  violations_24h: number;
}

export function usePolicyTimeBasedAccess() {
  const [data, setData] = useState<PolicyTimeBasedAccessData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      await new Promise((r) => setTimeout(r, 400));
      setData({
        time_window_rules: [
          { policy: "Production Access", allowed_days: ["Mon", "Tue", "Wed", "Thu", "Fri"], start_time: "08:00", end_time: "18:00", timezone: "America/New_York" },
          { policy: "Admin Console", allowed_days: ["Mon", "Tue", "Wed", "Thu", "Fri"], start_time: "09:00", end_time: "17:00", timezone: "America/New_York" },
          { policy: "DB Access", allowed_days: ["Mon", "Tue", "Wed", "Thu", "Fri", "Sat"], start_time: "07:00", end_time: "20:00", timezone: "UTC" },
        ],
        per_role_restrictions: [
          { role: "Developer", allowed_days: ["Mon", "Tue", "Wed", "Thu", "Fri"], start_time: "08:00", end_time: "19:00" },
          { role: "Admin", allowed_days: ["Mon", "Tue", "Wed", "Thu", "Fri"], start_time: "07:00", end_time: "18:00" },
        ],
        holiday_calendar: [
          { name: "New Year's Day", date: "2026-01-01", access: "blocked" },
          { name: "Memorial Day", date: "2026-05-25", access: "emergency_only" },
          { name: "Independence Day", date: "2026-07-04", access: "blocked" },
        ],
        grace_period_minutes: 15,
        violations_24h: 3,
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
