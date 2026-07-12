"use client";

import { useIdentityGroupMembershipAnalytics } from "@ggid/sdk-react";
import { Users, Layers, UserX, AlertTriangle, Sparkles, TrendingUp, Trash2 } from "lucide-react";

export default function IdentityGroupMembershipAnalyticsPage() {
  const { data, loading, error, refresh } = useIdentityGroupMembershipAnalytics();

  if (loading) return <div className="p-8 text-gray-400">Loading group membership analytics...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  const maxGrowth = Math.max(...(data?.membership_growth_30d ?? [1]), 1);

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">Group Membership Analytics</h1>
          <p className="text-sm text-gray-400 mt-1">Analyze group composition, growth, and detect shadow permissions</p>
        </div>
        <button
          onClick={refresh}
          className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition"
        >
          Refresh
        </button>
      </div>

      {/* Summary Stats */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-blue-400">
            <Users className="w-4 h-4" />
            <span className="text-xs text-gray-400">Total Groups</span>
          </div>
          <p className="text-2xl font-bold">{data?.group_cards?.length ?? 0}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-yellow-400">
            <UserX className="w-4 h-4" />
            <span className="text-xs text-gray-400">Inactive Members</span>
          </div>
          <p className="text-2xl font-bold">{data?.inactive_members?.length ?? 0}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-red-400">
            <AlertTriangle className="w-4 h-4" />
            <span className="text-xs text-gray-400">Orphaned Groups</span>
          </div>
          <p className="text-2xl font-bold">{data?.orphaned_groups?.length ?? 0}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-purple-400">
            <Sparkles className="w-4 h-4" />
            <span className="text-xs text-gray-400">Shadow Permissions</span>
          </div>
          <p className="text-2xl font-bold">{data?.shadow_permissions_detected ?? 0}</p>
        </div>
      </div>

      {/* Growth Chart + Recommendations */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6 mb-6">
        <div className="bg-gray-900 rounded-xl p-6 lg:col-span-2">
          <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
            <TrendingUp className="w-5 h-5 text-blue-400" />
            Membership Growth (30d)
          </h2>
          <div className="flex items-end gap-1 h-32">
            {(data?.membership_growth_30d ?? []).map((v, i) => (
              <div key={i} className="flex-1 flex flex-col items-center gap-1">
                <div
                  className="w-full rounded-t bg-blue-500 hover:bg-blue-400 transition-all"
                  style={{ height: `${(v / maxGrowth) * 100}%`, minHeight: "2px" }}
                  title={`Day ${i + 1}: ${v} members`}
                />
                {i % 7 === 0 && <span className="text-xs text-gray-500">{i + 1}</span>}
              </div>
            ))}
          </div>
        </div>

        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-sm font-semibold flex items-center gap-2 mb-3">
            <Sparkles className="w-4 h-4 text-cyan-400" />
            Cleanup Recommendations
          </h2>
          <div className="space-y-2">
            {(data?.recommend_cleanup ?? []).map((rec, i) => (
              <div key={i} className="bg-gray-800 rounded-lg p-3">
                <div className="flex items-center justify-between mb-1">
                  <p className="text-xs font-medium">{rec.action}</p>
                  <span
                    className={"text-xs px-2 py-0.5 rounded " + (
                      rec.priority === "high" ? "bg-red-900 text-red-300" :
                      rec.priority === "medium" ? "bg-yellow-900 text-yellow-300" :
                      "bg-blue-900 text-blue-300"
                    )}
                  >
                    {rec.priority}
                  </span>
                </div>
                <p className="text-xs text-gray-400">{rec.detail}</p>
              </div>
            ))}
          </div>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Group Cards */}
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-lg font-semibold mb-4">Group Overview</h2>
          <div className="space-y-2 max-h-80 overflow-y-auto">
            {(data?.group_cards ?? []).map((g) => (
              <div key={g.name} className="bg-gray-800 rounded-lg p-3">
                <div className="flex items-center justify-between mb-2">
                  <div className="flex items-center gap-2">
                    <Users className="w-4 h-4 text-blue-400" />
                    <p className="text-sm font-medium">{g.name}</p>
                  </div>
                  <span className="text-sm font-bold">{g.member_count}</span>
                </div>
                <div className="flex items-center gap-4 text-xs text-gray-400">
                  <span className="flex items-center gap-1">
                    <Layers className="w-3 h-3" />
                    Depth: {g.nesting_depth}
                  </span>
                  <span>Sub-groups: {g.sub_groups}</span>
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* Inactive Members + Orphaned Groups */}
        <div className="space-y-6">
          <div className="bg-gray-900 rounded-xl p-6">
            <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
              <UserX className="w-5 h-5 text-yellow-400" />
              Inactive Members
            </h2>
            <div className="space-y-2 max-h-40 overflow-y-auto">
              {(data?.inactive_members ?? []).map((m, i) => (
                <div key={i} className="flex items-center justify-between bg-gray-800 rounded-lg p-2">
                  <span className="text-sm text-gray-300">{m.user}</span>
                  <div className="flex items-center gap-2">
                    <span className="text-xs text-gray-400">{m.group}</span>
                    <span className={"text-xs px-2 py-0.5 rounded " + (
                      m.last_active_days > 90 ? "bg-red-900 text-red-300" : "bg-yellow-900 text-yellow-300"
                    )}>
                      {m.last_active_days}d inactive
                    </span>
                  </div>
                </div>
              ))}
              {(data?.inactive_members ?? []).length === 0 && (
                <p className="text-sm text-gray-500 text-center py-2">No inactive members.</p>
              )}
            </div>
          </div>

          <div className="bg-gray-900 rounded-xl p-6">
            <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
              <Trash2 className="w-5 h-5 text-red-400" />
              Orphaned Groups
            </h2>
            <div className="space-y-1">
              {(data?.orphaned_groups ?? []).map((g, i) => (
                <div key={i} className="flex items-center justify-between bg-gray-800 rounded-lg p-2">
                  <span className="text-sm text-gray-300">{g.name}</span>
                  <span className="text-xs text-gray-400">Last used: {g.last_used}</span>
                </div>
              ))}
              {(data?.orphaned_groups ?? []).length === 0 && (
                <p className="text-sm text-gray-500 text-center py-2">No orphaned groups.</p>
              )}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
