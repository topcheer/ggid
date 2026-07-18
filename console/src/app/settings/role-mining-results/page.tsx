"use client";

import { useRoleMiningResults } from "@ggid/sdk-react";
import { TrendingDown, Layers, Gauge } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

export default function RoleMiningResultsPage() {
  const t = useTranslations();
  const { data, loading, error, refresh } = useRoleMiningResults();
  if (loading) return <div className="p-8 text-gray-400">Loading role mining...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div><h1 className="text-2xl font-bold">Role Mining Results</h1><p className="text-sm text-gray-400 mt-1">Entitlement analysis and role optimization</p></div>
        <button onClick={refresh} className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition">Refresh</button>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-6">
        <div className="bg-gray-900 rounded-xl p-4 text-center"><Gauge className="w-5 h-5 text-yellow-400 mx-auto mb-1" /><p className="text-xs text-gray-400">Entitlement Creep Score</p><p className={"text-2xl font-bold " + ((data?.creep_score ?? 0) > 50 ? "text-red-400" : "text-green-400")}>{data?.creep_score ?? 0}</p></div>
        <div className="bg-gray-900 rounded-xl p-4 text-center"><TrendingDown className="w-5 h-5 text-red-400 mx-auto mb-1" /><p className="text-xs text-gray-400">Unused Permissions</p><p className="text-2xl font-bold">{data?.unused_permissions?.length ?? 0}</p></div>
        <div className="bg-gray-900 rounded-xl p-4 text-center"><Layers className="w-5 h-5 text-blue-400 mx-auto mb-1" /><p className="text-xs text-gray-400">Consolidation Suggestions</p><p className="text-2xl font-bold text-blue-400">{data?.suggested_consolidation?.length ?? 0}</p></div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-sm font-semibold mb-3">Unused Permissions</h2>
          <div className="space-y-2">
            {(data?.unused_permissions ?? []).map((u: any, i: number) => (
              <div key={i} className="flex items-center gap-2 bg-gray-800 rounded p-2 text-xs">
                <span className="font-mono text-gray-300 flex-1">{u.permission}</span>
                <span className="text-gray-400">{u.user}</span>
                <span className={"px-1.5 py-0.5 rounded " + (u.last_used_days > 90 ? "bg-red-900 text-red-300" : "bg-yellow-900 text-yellow-300")}>{u.last_used_days}d unused</span>
              </div>
            ))}
          </div>
        </div>
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-sm font-semibold mb-3">Suggested Consolidation</h2>
          <div className="space-y-2">
            {(data?.suggested_consolidation ?? []).map((c) => (
              <div key={c.merge_target} className="bg-gray-800 rounded-lg p-3">
                <p className="text-sm font-medium mb-1">Merge into: {c.merge_target}</p>
                <p className="text-xs text-gray-400">{c.roles_to_merge.join(", ")}</p>
                <p className="text-xs text-green-400 mt-1">Reduces {c.reduction_benefit} redundant assignments</p>
              </div>
            ))}
          </div>
        </div>
      </div>

      <div className="bg-gray-900 rounded-xl p-6 mt-6">
        <h2 className="text-sm font-semibold mb-3">Over-Assigned Roles</h2>
        <div className="space-y-1">
          {(data?.over_assigned ?? []).map((o) => (
            <div key={o.user} className="flex items-center gap-2 bg-gray-800 rounded p-2 text-xs">
              <span className="text-gray-300 flex-1">{o.user}</span>
              <span className="text-gray-400">{o.role}</span>
              <span className="text-red-400">{o.excess_permissions} excess perms</span>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
