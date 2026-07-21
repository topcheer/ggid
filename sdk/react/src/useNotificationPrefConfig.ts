import { useState, useCallback, useEffect } from "react";

/**
 * DEMO DATA — Tries real API first, falls back to empty demo data.
 * isDemoData flag indicates whether live or fallback data is shown.
 */

export interface MatrixRow {
  event: string;
  event_label: string;
  channels: string[];
}

export interface QuietHours {
  start: string;
  end: string;
  timezone: string;
}

export interface NotificationPrefConfigData {
  matrix: MatrixRow[];
  quiet_hours: QuietHours;
  digest_frequency: string;
  emergency_override: boolean;
}

export function useNotificationPrefConfig() {
  const [data, setData] = useState<NotificationPrefConfigData | null>(null);
  const [isDemoData, setIsDemoData] = useState(true);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const fetchData = useCallback(async () => {
    setLoading(true); setError(null);
    try { // Try real API first
      let res: Response | null = null;
      try {
        res = await fetch("/api/v1/data", { headers: { "Content-Type": "application/json" } });
      } catch { res = null; }
      if (res?.ok) { const d = await res.json(); setData(d); setIsDemoData(false); return; }
      setIsDemoData(true);
      setData({ matrix: [
        { event: "security_alert", event_label: "Security Alert", channels: ["email", "push", "webhook"] },
        { event: "access_change", event_label: "Access Change", channels: ["email"] },
        { event: "mfa_event", event_label: "MFA Event", channels: ["email", "push"] },
        { event: "compliance_deadline", event_label: "Compliance Deadline", channels: ["email", "webhook"] },
      ], quiet_hours: { start: "22:00", end: "07:00", timezone: "Asia/Shanghai" }, digest_frequency: "daily", emergency_override: true });
    } catch (e) { setError(e instanceof Error ? e.message : "Failed"); } finally { setLoading(false); }
  }, []);
  useEffect(() => { fetchData(); }, [fetchData]);
  return { data, loading, error, refresh: fetchData, isDemoData };
}
