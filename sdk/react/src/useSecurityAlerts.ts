import { useState, useCallback } from "react";

export interface SecurityAlert {
  id: string;
  type: string;
  severity: "low" | "medium" | "high" | "critical";
  source: string;
  timestamp: string;
  affected_users: number;
  detail: string;
  status: "active" | "acknowledged" | "resolved";
}

export function useSecurityAlerts(baseUrl: string = "") {
  const [alerts, setAlerts] = useState<SecurityAlert[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchAlerts = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(baseUrl + "/api/v1/audit/security-alerts");
      if (!res.ok) throw new Error("HTTP " + res.status);
      const data = await res.json(); setAlerts(data.alerts || data || []);
    } catch (e: any) { setError(e.message); }
    finally { setLoading(false); }
  }, [baseUrl]);

  const updateStatus = useCallback(async (id: string, status: string) => {
    try { await fetch(baseUrl + "/api/v1/audit/security-alerts/" + id, { method: "PATCH", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ status }) }); }
    catch { /* noop */ }
  }, [baseUrl]);

  return { alerts, loading, error, fetchAlerts, updateStatus };
}
