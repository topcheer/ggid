import { useState, useCallback, useEffect } from "react";

/**
 * DEMO DATA — Tries real API first, falls back to empty demo data.
 * isDemoData flag indicates whether live or fallback data is shown.
 */

export interface MonthlyMembership {
  month: string;
  joiners: number;
  leavers: number;
}

export interface DeptMembers {
  dept: string;
  members: number;
}

export interface AttritionReason {
  reason: string;
  count: number;
}

export interface MembershipTrendsData {
  retention_rate: number;
  net_growth_30d: number;
  avg_tenure_days: number;
  total_members: number;
  monthly: MonthlyMembership[];
  by_department: DeptMembers[];
  attrition_reasons: AttritionReason[];
}

export function useMembershipTrends() {
  const [data, setData] = useState<MembershipTrendsData | null>(null);
  const [isDemoData, setIsDemoData] = useState(true);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      // Try real API first
      let res: Response | null = null;
      try {
        res = await fetch("/api/v1/data", { headers: { "Content-Type": "application/json" } });
      } catch { res = null; }
      if (res?.ok) { const d = await res.json(); setData(d); setIsDemoData(false); return; }
      setIsDemoData(true);
      setData({
        retention_rate: 94.2,
        net_growth_30d: 42,
        avg_tenure_days: 365,
        total_members: 4833,
        monthly: [
          { month: "Jul", joiners: 45, leavers: 12 }, { month: "Aug", joiners: 52, leavers: 18 }, { month: "Sep", joiners: 38, leavers: 22 }, { month: "Oct", joiners: 61, leavers: 15 }, { month: "Nov", joiners: 33, leavers: 28 }, { month: "Dec", joiners: 28, leavers: 19 },
        ],
        by_department: [
          { dept: "Engineering", members: 1200 }, { dept: "Sales", members: 850 }, { dept: "Marketing", members: 420 }, { dept: "Operations", members: 680 }, { dept: "Finance", members: 280 },
        ],
        attrition_reasons: [
          { reason: "Career change", count: 45 }, { reason: "Relocation", count: 28 }, { reason: "Retirement", count: 15 }, { reason: "Performance", count: 12 },
        ],
      });
    } catch (e) { setError(e instanceof Error ? e.message : "Failed"); }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);
  return { data, loading, error, refresh: fetchData, isDemoData };
}
