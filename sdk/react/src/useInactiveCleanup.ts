import { useState, useCallback } from "react";

export interface InactiveUser {
  user_id: string;
  username: string;
  email: string;
  last_login: string | null;
  days_inactive: number;
  status: string;
}

export type CleanupAction = "disable" | "archive" | "delete";

export function useInactiveCleanup(baseUrl: string = "") {
  const [users, setUsers] = useState<InactiveUser[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchInactive = useCallback(async (thresholdDays: number = 90) => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/identity/inactive-users?threshold=${thresholdDays}`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json();
      setUsers(data.users || data || []);
    } catch (e: any) {
      setError(e.message);
    } finally {
      setLoading(false);
    }
  }, [baseUrl]);

  const scheduleCleanup = useCallback(async (userIds: string[], action: CleanupAction, scheduleDate?: string) => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/identity/inactive-cleanup/schedule`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ user_ids: userIds, action, schedule_date: scheduleDate }),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setUsers((prev) => prev.filter((u) => !userIds.includes(u.user_id)));
      return true;
    } catch (e: any) {
      setError(e.message);
      return false;
    } finally {
      setLoading(false);
    }
  }, [baseUrl]);

  return { users, loading, error, fetchInactive, scheduleCleanup };
}
