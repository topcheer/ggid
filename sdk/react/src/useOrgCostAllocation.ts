import { useState, useCallback, useEffect } from "react";

export interface AllocationRule {
  department: string;
  cost_center: string;
  allocation_pct: number;
  chargeback_model: "per_user" | "per_usage" | "fixed";
}

export interface MonthlyCost {
  department: string;
  amount: number;
}

export interface OverBudgetAlert {
  department: string;
  budget: number;
  actual: number;
  pct_over: number;
}

export interface OrgCostAllocationData {
  allocation_rules: AllocationRule[];
  monthly_cost_breakdown: MonthlyCost[];
  chargeback_report_preview: Record<string, unknown>;
  over_budget_alerts: OverBudgetAlert[];
}

export function useOrgCostAllocation() {
  const [data, setData] = useState<OrgCostAllocationData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      await new Promise((r) => setTimeout(r, 400));
      setData({
        allocation_rules: [
          { department: "Engineering", cost_center: "CC-1001", allocation_pct: 40, chargeback_model: "per_user" },
          { department: "Sales", cost_center: "CC-2002", allocation_pct: 25, chargeback_model: "per_usage" },
          { department: "Finance", cost_center: "CC-3003", allocation_pct: 15, chargeback_model: "fixed" },
          { department: "Marketing", cost_center: "CC-4004", allocation_pct: 12, chargeback_model: "per_user" },
          { department: "Operations", cost_center: "CC-5005", allocation_pct: 8, chargeback_model: "fixed" },
        ],
        monthly_cost_breakdown: [
          { department: "Engineering", amount: 48000 },
          { department: "Sales", amount: 30000 },
          { department: "Finance", amount: 18000 },
          { department: "Marketing", amount: 14400 },
          { department: "Operations", amount: 9600 },
        ],
        chargeback_report_preview: {
          billing_period: "2026-01",
          total_cost: 120000,
          currency: "USD",
          departments: 5,
          model: "hybrid (per_user + fixed)",
          generated_at: "2026-01-15T00:00:00Z",
        },
        over_budget_alerts: [
          { department: "Sales", budget: 25000, actual: 30000, pct_over: 20 },
          { department: "Marketing", budget: 12000, actual: 14400, pct_over: 20 },
        ],
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
