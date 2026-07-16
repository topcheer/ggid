"use client";

import { useState, useEffect, useCallback } from "react";
import { Building2, Users, TrendingUp, TrendingDown } from "lucide-react";
import { useTranslations } from "@/lib/i18n";
import { authHeader, isAuthenticated } from "@/lib/auth-helpers";

interface Dept {
  id: string;
  name: string;
  headcount: number;
  avg_tenure_days: number;
  growth_rate_30d: number;
  budget_utilization_pct: number;
  open_positions: number;
  attrition_rate: number;
}

export default function DepartmentAnalyticsPage() {
  const t = useTranslations();

  const [depts, setDepts] = useState<Dept[]>([]);
  const [loading, setLoading] = useState(false);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/v1/org/department-analytics", { headers: { ...authHeader(), "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) { const d = await res.json(); setDepts(d.departments || d || []); }
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  const totalHeadcount = depts.reduce((s, d) => s + d.headcount, 0);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><Building2 className="w-6 h-6 text-blue-500" /> {t("departmentAnalytics.title")}</h1>
        <p className="text-sm text-gray-500 mt-1">Workforce analytics across departments with tenure, growth, and attrition.</p>
      </div>

      <div className="grid grid-cols-3 gap-4">
        <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">{t("backend3.departmentAnalytics.departments")}</span><p className="text-xl font-bold mt-1">{depts.length}</p></div>
        <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">{t("backend3.departmentAnalytics.totalHeadcount")}</span><p className="text-xl font-bold mt-1">{totalHeadcount}</p></div>
        <div className="rounded-lg border p-4 dark:border-gray-800"><span className="text-sm text-gray-500">{t("backend3.departmentAnalytics.openPositions")}</span><p className="text-xl font-bold text-orange-600 mt-1">{depts.reduce((s, d) => s + d.open_positions, 0)}</p></div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {depts.map((d) => (
          <div key={d.id} className="rounded-lg border dark:border-gray-800 p-4 space-y-3">
            <div className="flex items-center justify-between"><div className="flex items-center gap-2"><Building2 className="w-5 h-5 text-gray-400" /><span className="font-semibold">{d.name}</span></div><span className="flex items-center gap-1 text-sm text-gray-500"><Users className="w-3.5 h-3.5" />{d.headcount}</span></div>
            <div className="grid grid-cols-2 gap-2 text-sm">
              <div><span className="text-xs text-gray-500">{t("backend3.departmentAnalytics.avgTenure")}</span><p className="font-medium">{Math.round(d.avg_tenure_days / 30)}mo</p></div>
              <div><span className="text-xs text-gray-500">Growth 30d</span><p className={"font-medium flex items-center gap-1 " + (d.growth_rate_30d >= 0 ? "text-green-600" : "text-red-600")}>{d.growth_rate_30d >= 0 ? <TrendingUp className="w-3 h-3" /> : <TrendingDown className="w-3 h-3" />}{Math.abs(d.growth_rate_30d).toFixed(1)}%</p></div>
              <div><span className="text-xs text-gray-500">{t("backend3.departmentAnalytics.budgetUtil")}</span><div className="flex items-center gap-1"><div className="w-16 bg-gray-100 dark:bg-gray-800 rounded-full h-2 overflow-hidden"><div className={"h-full rounded-full " + (d.budget_utilization_pct > 90 ? "bg-red-500" : d.budget_utilization_pct > 75 ? "bg-yellow-500" : "bg-green-500")} style={{ width: d.budget_utilization_pct + "%" }} /></div><span className="text-xs font-bold">{d.budget_utilization_pct.toFixed(0)}%</span></div></div>
              <div><span className="text-xs text-gray-500">{t("backend3.departmentAnalytics.attrition")}</span><p className={"font-medium " + (d.attrition_rate > 10 ? "text-red-600" : d.attrition_rate > 5 ? "text-yellow-600" : "text-green-600")}>{d.attrition_rate.toFixed(1)}%</p></div>
            </div>
            {d.open_positions > 0 && <div className="text-xs text-orange-600 border-t dark:border-gray-800 pt-2">{d.open_positions} open position{d.open_positions > 1 ? "s" : ""}</div>}
          </div>
        ))}
        {depts.length === 0 && !loading && <div className="col-span-full text-center text-gray-500 py-8">No departments found.</div>}
      </div>
    </div>
  );
}
