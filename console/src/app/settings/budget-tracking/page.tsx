"use client";

import { useState, useEffect, useCallback } from "react";
import { DollarSign, TrendingUp, TrendingDown, AlertTriangle } from "lucide-react";

interface DeptBudget {
  department: string;
  spent: number;
  budget: number;
  burn_rate: number;
  projected_eoy: number;
  cost_per_user: number;
  member_count: number;
}

interface BudgetData {
  departments: DeptBudget[];
  total_spent: number;
  total_budget: number;
}

export default function BudgetTrackingPage() {
  const [data, setData] = useState<BudgetData | null>(null);
  const [loading, setLoading] = useState(false);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/v1/identity/budget-tracking", { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) setData(await res.json());
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  const fmt = (n: number) => "$" + n.toLocaleString(undefined, { maximumFractionDigits: 0 });

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><DollarSign className="w-6 h-6 text-green-500" /> Budget Tracking</h1>
        <p className="text-sm text-gray-500 mt-1">Track departmental spending with burn rate and end-of-year projections.</p>
      </div>

      {data && (
        <>
          <div className="grid grid-cols-3 gap-4">
            <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">Total Spent</span><p className="text-xl font-bold mt-1">{fmt(data.total_spent)}</p></div>
            <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">Total Budget</span><p className="text-xl font-bold mt-1">{fmt(data.total_budget)}</p></div>
            <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">Remaining</span><p className={`text-xl font-bold mt-1 ${data.total_budget - data.total_spent < 0 ? "text-red-600" : "text-green-600"}`}>{fmt(data.total_budget - data.total_spent)}</p></div>
          </div>

          <div className="space-y-3">{data.departments.map((d) => {
            const pct = d.budget > 0 ? (d.spent / d.budget) * 100 : 0;
            const isOver = pct >= 100;
            const isWarning = pct >= 80;
            const projectedOver = d.projected_eoy > d.budget;
            return (
              <div key={d.department} className="rounded-lg border dark:border-gray-800 p-4 space-y-3">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2"><span className="font-semibold">{d.department}</span>{isOver && <span className="px-2 py-0.5 rounded text-xs bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400 flex items-center gap-1"><AlertTriangle className="w-3 h-3" /> Over Budget</span>}{!isOver && isWarning && <span className="px-2 py-0.5 rounded text-xs bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400">{pct.toFixed(0)}% used</span>}</div>
                  <span className="text-sm font-bold">{fmt(d.spent)} / {fmt(d.budget)}</span>
                </div>
                <div className="w-full bg-gray-100 dark:bg-gray-800 rounded-full h-3 overflow-hidden"><div className="h-full rounded-full transition-all" style={{ width: `${Math.min(pct, 100)}%`, background: isOver ? "#ef4444" : isWarning ? "#f59e0b" : "#10b981" }} /></div>
                <div className="grid grid-cols-4 gap-2 text-sm">
                  <div><span className="text-gray-500 text-xs">Burn Rate</span><p className="font-medium">{fmt(d.burn_rate)}/mo</p></div>
                  <div><span className="text-gray-500 text-xs">Projected EOY</span><p className={`font-medium ${projectedOver ? "text-red-600" : ""}`}>{fmt(d.projected_eoy)}</p></div>
                  <div><span className="text-gray-500 text-xs">Cost/User</span><p className="font-medium">{fmt(d.cost_per_user)}</p></div>
                  <div><span className="text-gray-500 text-xs">Users</span><p className="font-medium">{d.member_count}</p></div>
                </div>
              </div>
            );
          })}</div>
        </>
      )}
      {!data && !loading && <p className="text-sm text-gray-500 text-center py-8">Loading...</p>}
    </div>
  );
}
