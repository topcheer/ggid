import { useState, useCallback } from "react";

export interface DeptBudget {
  department: string;
  spent: number;
  budget: number;
  burn_rate: number;
  projected_eoy: number;
  cost_per_user: number;
  member_count: number;
}

export interface BudgetData {
  departments: DeptBudget[];
  total_spent: number;
  total_budget: number;
}

export function useBudgetTracking(baseUrl: string = "") {
  const [data, setData] = useState<BudgetData | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchBudget = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/identity/budget-tracking`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setData(await res.json());
    } catch (e: any) { setError(e.message); }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { data, loading, error, fetchBudget };
}
