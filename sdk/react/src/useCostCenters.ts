import { useState, useCallback } from "react";

export interface Department {
  id: string;
  name: string;
  cost_center: string;
  member_count: number;
  resource_usage: number;
  budget: number;
  budget_used_pct: number;
  alerts: string[];
}

export interface CostData {
  departments: Department[];
  allocation: { department: string; amount: number }[];
}

export function useCostCenters(baseUrl: string = "") {
  const [data, setData] = useState<CostData | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchCostCenters = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(`${baseUrl}/api/v1/identity/cost-centers`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setData(await res.json());
    } catch (e: any) { setError(e.message); }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { data, loading, error, fetchCostCenters };
}
