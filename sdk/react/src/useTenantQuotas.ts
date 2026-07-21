import { useState, useCallback, useEffect } from "react";

/**
 * DEMO DATA — Tries real API first, falls back to empty demo data.
 */

export interface UsageItem {
  resource: string;
  used: number;
  limit: number;
}

export interface PlanLimit {
  plan: string;
  limits: Record<string, number>;
}

export interface OverageAlert {
  resource: string;
  severity: "warning" | "critical";
  message: string;
}

export interface TenantQuotasData {
  current_plan: string;
  days_until_reset: number;
  usage: UsageItem[];
  per_plan_limits: PlanLimit[];
  overage_alerts: OverageAlert[];
  usage_trend_30d: number[];
}

export function useTenantQuotas() {
  const [data, setData] = useState<TenantQuotasData | null>(null);
  const [isDemoData, setIsDemoData] = useState(true);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      // Try real API first
      let res: Response | null = null;
      try { res = await fetch("/api/v1/data", { headers: { "Content-Type": "application/json" } }); } catch { res = null; }
      if (res?.ok) { const d = await res.json(); setData(d); setIsDemoData(false); return; }
      setIsDemoData(true);
      setData({
        current_plan: "pro",
        days_until_reset: 12,
        usage: [
          { resource: "users", used: 842, limit: 1000 },
          { resource: "api_calls", used: 485000, limit: 500000 },
          { resource: "storage_mb", used: 12400, limit: 50000 },
          { resource: "sessions", used: 320, limit: 500 },
          { resource: "tokens_issued", used: 12500, limit: 50000 },
        ],
        per_plan_limits: [
          { plan: "free", limits: { users: 50, api_calls: 10000, storage_mb: 500, sessions: 25, tokens_issued: 1000 } },
          { plan: "starter", limits: { users: 200, api_calls: 50000, storage_mb: 5000, sessions: 100, tokens_issued: 5000 } },
          { plan: "pro", limits: { users: 1000, api_calls: 500000, storage_mb: 50000, sessions: 500, tokens_issued: 50000 } },
          { plan: "enterprise", limits: { users: -1, api_calls: -1, storage_mb: -1, sessions: -1, tokens_issued: -1 } },
        ],
        overage_alerts: [
          { resource: "api_calls", severity: "warning", message: "API call usage at 97% of limit" },
          { resource: "sessions", severity: "warning", message: "Concurrent sessions at 64% of limit" },
        ],
        usage_trend_30d: Array.from({ length: 30 }, (_, i) => Math.round(20000 + Math.sin(i / 4) * 8000 + i * 500)),
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

  return { data, loading, error, refresh: fetchData, isDemoData };
}
