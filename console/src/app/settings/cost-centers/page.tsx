"use client";

import { useState, useEffect, useCallback } from "react";
import { Building2, DollarSign, AlertTriangle, Users } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface Department {
  id: string;
  name: string;
  cost_center: string;
  member_count: number;
  resource_usage: number;
  budget: number;
  budget_used_pct: number;
  alerts: string[];
}

interface CostData {
  departments: Department[];
  allocation: { department: string; amount: number }[];
}

const allocColors = ["#3b82f6", "#8b5cf6", "#10b981", "#f59e0b", "#ef4444", "#06b6d4"];

export default function CostCentersPage() {
  const t = useTranslations();

  const [data, setData] = useState<CostData | null>(null);
  const [loading, setLoading] = useState(false);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/v1/identity/cost-centers", { headers: { "Authorization": `Bearer ${localStorage.getItem("ggid_access_token") || ""}`, "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) setData(await res.json());
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  const totalAlloc = data?.allocation.reduce((s, a) => s + a.amount, 0) || 1;
  const totalBudget = data?.departments.reduce((s, d) => s + d.budget, 0) || 0;
  const totalUsed = data?.departments.reduce((s, d) => s + (d.budget * d.budget_used_pct / 100), 0) || 0;

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><DollarSign className="w-6 h-6 text-green-500" /> {t("costCenters.title")}</h1>
        <p className="text-sm text-gray-500 mt-1">Track department-level cost center allocations and budget usage.</p>
      </div>

      {data && (
        <>
          <div className="grid grid-cols-1 lg:grid-cols-4 gap-4">
            <div className="rounded-lg border dark:border-gray-800 p-4 flex items-center gap-3"><Building2 className="w-8 h-8 text-blue-500" /><div><span className="text-sm text-gray-500">{t("backend3.costCenters.departments")}</span><p className="text-xl font-bold mt-1">{data.departments.length}</p></div></div>
            <div className="rounded-lg border dark:border-gray-800 p-4 flex items-center gap-3"><DollarSign className="w-8 h-8 text-green-500" /><div><span className="text-sm text-gray-500">{t("backend3.costCenters.totalBudget")}</span><p className="text-xl font-bold mt-1">${totalBudget.toLocaleString()}</p></div></div>
            <div className="rounded-lg border dark:border-gray-800 p-4 flex items-center gap-3"><DollarSign className="w-8 h-8 text-orange-500" /><div><span className="text-sm text-gray-500">{t("backend3.costCenters.used")}</span><p className="text-xl font-bold text-orange-600 mt-1">${totalUsed.toLocaleString(undefined, { maximumFractionDigits: 0 })}</p></div></div>
            <div className="rounded-lg border dark:border-gray-800 p-4">
              <span className="text-sm text-gray-500">{t("backend3.costCenters.allocation")}</span>
              <div className="flex items-center gap-2 mt-2"><svg viewBox="0 0 64 64" className="w-12 h-12 -rotate-90">{(() => { let off = 0; return data.allocation.map((a, i) => { const pct = a.amount / totalAlloc; const dash = pct * 176; const c = <circle key={i} cx={32} cy={32} r={28} fill="none" stroke={allocColors[i % allocColors.length]} strokeWidth={8} strokeDasharray={`${dash} 176`} strokeDashoffset={-off * 176} />; off += pct; return c; }); })()}</svg></div>
            </div>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {data.departments.map((d) => (
              <div key={d.id} className="rounded-lg border dark:border-gray-800 p-4 space-y-3">
                <div className="flex items-center justify-between"><div><span className="font-semibold">{d.name}</span><p className="text-xs font-mono text-gray-400">{d.cost_center}</p></div>{d.alerts.length > 0 && <span className="px-2 py-0.5 rounded text-xs bg-red-100 dark:bg-red-900/30 dark:text-red-400 flex items-center gap-1"><AlertTriangle className="w-3 h-3" />{d.alerts.length}</span>}</div>
                <div className="grid grid-cols-2 gap-2 text-sm"><div className="flex items-center gap-1"><Users className="w-3.5 h-3.5 text-gray-400" /><span className="text-gray-500">{t("backend3.costCenters.members")}</span><span className="font-bold ml-auto">{d.member_count}</span></div><div className="flex items-center gap-1"><DollarSign className="w-3.5 h-3.5 text-gray-400" /><span className="text-gray-500">{t("backend3.costCenters.budget")}</span><span className="font-bold ml-auto">${d.budget.toLocaleString()}</span></div></div>
                <div><div className="flex items-center justify-between text-xs mb-1"><span className="text-gray-500">{t("backend3.costCenters.resourceUsage")}</span><span className={`font-bold ${d.budget_used_pct >= 90 ? "text-red-600" : d.budget_used_pct >= 70 ? "text-orange-600" : "text-green-600"}`}>{d.budget_used_pct.toFixed(0)}%</span></div><div className="w-full bg-gray-100 dark:bg-gray-800 rounded-full h-2 overflow-hidden"><div className="h-full rounded-full" style={{ width: `${d.budget_used_pct}%`, background: d.budget_used_pct >= 90 ? "#ef4444" : d.budget_used_pct >= 70 ? "#f59e0b" : "#10b981" }} /></div></div>
                {d.alerts.length > 0 && <div className="space-y-1">{d.alerts.map((a, i) => <div key={i} className="text-xs text-red-500">- {a}</div>)}</div>}
              </div>
            ))}
          </div>
        </>
      )}
      {!data && !loading && <p className="text-sm text-gray-500 text-center py-8">Loading...</p>}
    </div>
  );
}
