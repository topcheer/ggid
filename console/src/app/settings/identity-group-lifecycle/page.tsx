"use client";

import { useIdentityGroupLifecycle } from "@ggid/sdk-react";
import { Users, Archive, Activity, GitMerge, Download, AlertTriangle } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

export default function IdentityGroupLifecyclePage() {
  const { data, loading, error, refresh } = useIdentityGroupLifecycle();
  const t = useTranslations();

  if (loading) return <div className="p-8 text-gray-400">{t("idGroupLifecycle.loading")}</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  const statusColors: Record<string, string> = {
    active: "#22c55e",
    dormant: "#eab308",
    empty: "#6b7280",
    deprecated: "#ef4444",
  };

  const statusData = data?.groups_by_status ?? { active: 0, dormant: 0, empty: 0, deprecated: 0 };
  const statusEntries: [string, number][] = Object.entries(statusData) as [string, number][];
  const total = statusEntries.reduce((a, [, c]) => a + c, 0);

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">{t("idGroupLifecycle.title")}</h1>
          <p className="text-sm text-gray-400 mt-1">{t("idGroupLifecycle.subtitle")}</p>
        </div>
        <div className="flex items-center gap-2">
          <button className="flex items-center gap-1 px-3 py-2 bg-gray-700 hover:bg-gray-600 rounded-lg text-sm font-medium">
            <Download className="w-4 h-4" /> Export
          </button>
          <button onClick={refresh} className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition">{t("idGroupLifecycle.refresh")}</button>
        </div>
      </div>

      {/* Summary + Donut */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6 mb-6">
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-sm font-semibold mb-4">{t("idGroupLifecycle.groupsByStatus")}</h2>
          <div className="relative w-32 h-32 mx-auto">
            <svg className="w-32 h-32 -rotate-90" viewBox="0 0 100 100">
              {(() => {
                let offset = 0;
                return statusEntries.map(([status, count]) => {
                  const pct = total > 0 ? count / total : 0;
                  const dash = pct * 251.2;
                  const el = (
                    <circle
                      key={status}
                      cx="50" cy="50" r="40"
                      fill="none"
                      stroke={statusColors[status] ?? "#6b7280"}
                      strokeWidth="12"
                      strokeDasharray={dash + " " + (251.2 - dash)}
                      strokeDashoffset={-offset}
                    />
                  );
                  offset += dash;
                  return el;
                });
              })()}
            </svg>
            <div className="absolute inset-0 flex items-center justify-center">
              <span className="text-2xl font-bold">{total}</span>
            </div>
          </div>
          <div className="mt-4 space-y-1">
            {statusEntries.map(([status, count]) => (
              <div key={status} className="flex items-center gap-2 text-xs">
                <span className="w-2 h-2 rounded-full" style={{ backgroundColor: statusColors[status] ?? "#6b7280" }} />
                <span className="capitalize text-gray-400">{status}</span>
                <span className="font-medium ml-auto">{count as number}</span>
              </div>
            ))}
          </div>
        </div>

        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-sm font-semibold flex items-center gap-2 mb-3">
            <Archive className="w-4 h-4 text-yellow-400" />
            Auto-Archive
          </h2>
          <p className="text-3xl font-bold mb-1">{data?.auto_archive_after_days ?? 0}</p>
          <p className="text-xs text-gray-400">Days of inactivity before auto-archive</p>
        </div>

        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-sm font-semibold flex items-center gap-2 mb-3">
            <GitMerge className="w-4 h-4 text-blue-400" />
            Merge Wizard
          </h2>
          <p className="text-xs text-gray-400 mb-3">{t("idGroupLifecycle.consolidate")}</p>
          <button className="w-full px-3 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition">
            Start Merge Wizard
          </button>
        </div>
      </div>

      {/* Group Health Metrics */}
      <div className="bg-gray-900 rounded-xl p-6 mb-6">
        <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
          <Activity className="w-5 h-5 text-purple-400" />
          Group Health Metrics
        </h2>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-gray-800 text-gray-400">
                <th className="text-left py-2 pr-3">{t("idGroupLifecycle.group")}</th>
                <th className="text-left py-2 pr-3">{t("idGroupLifecycle.members")}</th>
                <th className="text-left py-2 pr-3">{t("idGroupLifecycle.activityScore")}</th>
                <th className="text-left py-2 pr-3">{t("idGroupLifecycle.permissionFreshness")}</th>
                <th className="text-left py-2 pr-3">{t("idGroupLifecycle.status")}</th>
              </tr>
            </thead>
            <tbody>
              {(data?.group_health_metrics ?? []).map((g) => (
                <tr key={g.group_name} className="border-b border-gray-800">
                  <td className="py-3 pr-3">
                    <div className="flex items-center gap-2">
                      <Users className="w-3 h-3 text-gray-500" />
                      <span className="text-sm font-medium">{g.group_name}</span>
                    </div>
                  </td>
                  <td className="py-3 pr-3 text-gray-400">{g.member_count}</td>
                  <td className="py-3 pr-3">
                    <div className="flex items-center gap-2">
                      <div className="w-16 h-1.5 bg-gray-700 rounded-full">
                        <div className={"h-full rounded-full " + (g.member_activity_score > 0.7 ? "bg-green-500" : g.member_activity_score > 0.3 ? "bg-yellow-500" : "bg-red-500")} style={{ width: (g.member_activity_score * 100) + "%" }} />
                      </div>
                      <span className="text-xs">{(g.member_activity_score * 100).toFixed(0)}%</span>
                    </div>
                  </td>
                  <td className="py-3 pr-3 text-xs">
                    <span className={g.permission_freshness < 30 ? "text-red-400" : g.permission_freshness < 90 ? "text-yellow-400" : "text-green-400"}>
                      {g.permission_freshness}d ago
                    </span>
                  </td>
                  <td className="py-3 pr-3">
                    <span className={"text-xs px-2 py-0.5 rounded " + (
                      g.status === "active" ? "bg-green-900 text-green-300" :
                      g.status === "dormant" ? "bg-yellow-900 text-yellow-300" :
                      g.status === "deprecated" ? "bg-red-900 text-red-300" :
                      "bg-gray-700 text-gray-400"
                    )}>
                      {g.status}
                    </span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      {/* Cleanup Recommendations */}
      <div className="bg-gray-900 rounded-xl p-6">
        <h2 className="text-sm font-semibold flex items-center gap-2 mb-4">
          <AlertTriangle className="w-4 h-4 text-yellow-400" />
          Cleanup Recommendations
        </h2>
        <div className="space-y-2">
          {(data?.cleanup_recommendations ?? []).map((r, i) => (
            <div key={i} className="flex items-center gap-3 bg-gray-800 rounded-lg p-3">
              <span className={"w-2 h-2 rounded-full " + (
                r.priority === "high" ? "bg-red-500" :
                r.priority === "medium" ? "bg-yellow-500" :
                "bg-green-500"
              )} />
              <div className="flex-1">
                <p className="text-sm font-medium">{r.action}</p>
                <p className="text-xs text-gray-400">{r.group_name} - {r.reason}</p>
              </div>
              <span className="text-xs capitalize text-gray-500">{r.priority}</span>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
