import { useState, useCallback } from "react";

export interface Department {
  id: string;
  name: string;
  headcount: number;
  avg_tenure_days: number;
  growth_rate_30d: number;
  budget_utilization_pct: number;
  open_positions: number;
  attrition_rate: number;
}

export function useDepartmentAnalytics(baseUrl: string = "") {
  const [departments, setDepartments] = useState<Department[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchDepartments = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const res = await fetch(baseUrl + "/api/v1/org/department-analytics");
      if (!res.ok) throw new Error("HTTP " + res.status);
      const data = await res.json(); setDepartments(data.departments || data || []);
    } catch (e: any) { setError(e.message); }
    finally { setLoading(false); }
  }, [baseUrl]);

  return { departments, loading, error, fetchDepartments };
}
