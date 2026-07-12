"use client";

import { useMembershipTrends } from "@ggid/sdk-react";
import { TrendingUp, TrendingDown } from "lucide-react";

export default function MembershipTrendsPage() {
  const { data, loading, error, refresh } = useMembershipTrends();

  if (loading) return <div className="p-8 text-gray-400">Loading membership trends...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div><h1 className="text-2xl font-bold">Membership Trends</h1><p className="text-sm text-gray-400 mt-1">Joiners, leavers, and retention analytics</p></div>
        <button onClick={refresh} className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition">Refresh</button>
      </div>

      {/* Summary Cards */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
        <div className="bg-gray-900 rounded-xl p-4"><TrendingUp className="w-5 h-5 text-green-400 mb-1" /><p className="text-xs text-gray-400">Retention Rate</p><p className="text-xl font-bold text-green-400">{data?.retention_rate ?? 0}%</p></div>
        <div className="bg-gray-900 rounded-xl p-4"><p className="text-xs text-gray-400">Net Growth (30d)</p><p className={"text-xl font-bold " + ((data?.net_growth_30d ?? 0) >= 0 ? "text-green-400" : "text-red-400")}>{data?.net_growth_30d ?? 0 >= 0 ? "+" : ""}{data?.net_growth_30d ?? 0}</p></div>
        <div className="bg-gray-900 rounded-xl p-4"><p className="text-xs text-gray-400">Avg Tenure</p><p className="text-xl font-bold">{data?.avg_tenure_days ?? 0}d</p></div>
        <div className="bg-gray-900 rounded-xl p-4"><p className="text-xs text-gray-400">Total Members</p><p className="text-xl font-bold">{(data?.total_members ?? 0).toLocaleString()}</p></div>
      </div>

      {/* Monthly Bar Chart */}
      <div className="bg-gray-900 rounded-xl p-6 mb-6">
        <h2 className="text-sm font-semibold mb-4">Joiners vs Leavers (12 months)</h2>
        <div className="flex items-end gap-2 h-40">
          {(data?.monthly ?? []).map((m) => {
            const max = Math.max(...(data?.monthly ?? []).map((x) => Math.max(x.joiners, x.leavers)), 1);
            return (
              <div key={m.month} className="flex-1 flex flex-col items-center">
                <div className="flex items-end gap-0.5 w-full justify-center">
                  <div className="w-3 bg-green-600 rounded-t" style={{ height: (m.joiners / max * 100) + "px" }} title={m.joiners + " joiners"} />
                  <div className="w-3 bg-red-600 rounded-t" style={{ height: (m.leavers / max * 100) + "px" }} title={m.leavers + " leavers"} />
                </div>
                <span className="text-xs text-gray-500 mt-1">{m.month}</span>
              </div>
            );
          })}
        </div>
        <div className="flex items-center gap-4 mt-3 text-xs"><span className="flex items-center gap-1"><span className="w-3 h-3 bg-green-600 rounded" /> Joiners</span><span className="flex items-center gap-1"><span className="w-3 h-3 bg-red-600 rounded" /> Leavers</span></div>
      </div>

      {/* Department + Attrition */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-sm font-semibold mb-3">By Department</h2>
          <div className="space-y-2">
            {(data?.by_department ?? []).map((d) => (
              <div key={d.dept} className="flex items-center gap-2"><span className="text-xs w-24">{d.dept}</span><div className="flex-1 bg-gray-800 rounded-full h-2"><div className="bg-blue-600 h-2 rounded-full" style={{ width: (d.members / (data?.total_members ?? 1) * 100) + "%" }} /></div><span className="text-xs text-gray-400">{d.members}</span></div>
            ))}
          </div>
        </div>
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-sm font-semibold mb-3">Top Attrition Reasons</h2>
          <div className="space-y-2">
            {(data?.attrition_reasons ?? []).map((r) => (
              <div key={r.reason} className="flex items-center gap-2"><TrendingDown className="w-3 h-3 text-red-400" /><span className="text-xs flex-1">{r.reason}</span><span className="text-xs text-gray-400">{r.count}</span></div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}
