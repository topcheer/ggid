import { useState, useCallback } from "react";

export interface LifecycleUser {
  id: string;
  username: string;
  stage: "active" | "dormant" | "suspended" | "deactivated" | "pending";
  last_active: string;
  days_inactive: number;
  stage_since: string;
}

export function useUserLifecycle(baseUrl: string = "") {
  const [users, setUsers] = useState<LifecycleUser[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchUsers = useCallback(async (stage: string) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(baseUrl + "/api/v1/identity/user-lifecycle?stage=" + stage);
      if (!res.ok) throw new Error("HTTP " + res.status);
      const data = await res.json(); setUsers(data.users || data || []);
    } catch (e: any) { setError(e.message); }
    finally { setLoading(false); }
  }, [baseUrl]);

  const bulkAction = useCallback(async (action: string, stage: string, userIds: string[]) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(baseUrl + "/api/v1/identity/user-lifecycle/bulk", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ action, stage, user_ids: userIds }) });
      if (!res.ok) throw new Error("HTTP " + res.status);
      return true;
    } catch (e: any) { setError(e.message); return false; }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { users, loading, error, fetchUsers, bulkAction };
}
