import { useState, useCallback } from "react";

export interface RotationItem {
  id: string;
  credential_type: string;
  user_id: string;
  username: string;
  last_rotated: string;
  rotation_period_days: number;
  days_overdue: number;
  severity: "low" | "medium" | "high" | "critical";
}

export function useRotationReminders(baseUrl: string = "") {
  const [items, setItems] = useState<RotationItem[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchItems = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/auth/rotation-reminders`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json(); setItems(data.items || data || []);
    } catch (e: any) { setError(e.message); }
    finally { setLoading(false); }
  }, [baseUrl]);

  const sendReminder = useCallback(async (id: string) => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/auth/rotation-reminders/${id}/send`, { method: "POST" });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      return true;
    } catch (e: any) { setError(e.message); return false; }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { items, loading, error, fetchItems, sendReminder };
}
