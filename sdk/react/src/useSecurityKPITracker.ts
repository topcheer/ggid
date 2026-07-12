import { useState, useCallback, useEffect } from "react";

export interface KpiDefinition {
  name: string;
  target: number;
  current: number;
  unit: string;
  trend: string;
  owner: string;
}

export interface MonthlyPoint {
  month: string;
  value: number;
}

export interface AlertThreshold {
  kpi: string;
  threshold: string;
  triggered: boolean;
}

export interface SecurityKPITrackerData {
  kpi_definitions: KpiDefinition[];
  monthly_history: MonthlyPoint[];
  alert_thresholds: AlertThreshold[];
}

export function useSecurityKPITracker() {
  const [data, setData] = useState<SecurityKPITrackerData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      await new Promise((r) => setTimeout(r, 400));
      setData({
        kpi_definitions: [
          { name: "Critical vuln remediation SLA", target: 7, current: 5, unit: "days", trend: "down", owner: "sec-ops" },
          { name: "Phishing simulation click rate", target: 5, current: 3, unit: "%", trend: "down", owner: "sec-awareness" },
          { name: "MFA enrollment", target: 100, current: 94, unit: "%", trend: "up", owner: "identity-team" },
          { name: "Patch lead time", target: 14, current: 18, unit: "days", trend: "up", owner: "infra" },
          { name: "Security training completion", target: 90, current: 87, unit: "%", trend: "up", owner: "hr-sec" },
        ],
        monthly_history: [
          { month: "Jul", value: 72 }, { month: "Aug", value: 75 }, { month: "Sep", value: 73 }, { month: "Oct", value: 78 }, { month: "Nov", value: 80 }, { month: "Dec", value: 82 },
        ],
        alert_thresholds: [
          { kpi: "Patch lead time", threshold: "14d", triggered: true },
          { kpi: "MFA enrollment", threshold: "95%", triggered: false },
          { kpi: "Critical vuln SLA", threshold: "7d", triggered: false },
        ],
      });
    } catch (e) { setError(e instanceof Error ? e.message : "Failed"); }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);
  return { data, loading, error, refresh: fetchData };
}
