"use client";

import { useTranslations } from "@/lib/i18n";
import { usePolicyLifecycleDashboard } from "@ggid/sdk-react";
import { RotateCcw, Clock, GitCommit, PieChart } from "lucide-react";

export default function PolicyLifecycleDashboardPage() {
  const { data, loading, error, refresh } = usePolicyLifecycleDashboard();
  const t = useTranslations();

  if (loading) return <div className="p-8 text-gray-400">{t("policyLifecycle.loading")}</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  const statusColors: Record<string, string> = {
    draft: "#6366f1",
    active: "#22c55e",
    quarantined: "#eab308",
    deprecated: "#ef4444",
  };

  const statusData = data?.policies_by_status ?? { draft: 0, active: 0, quarantined: 0, deprecated: 0 };
  const statusEntries: [string, number][] = Object.entries(statusData) as [string, number][];
  const totalPolicies = statusEntries.reduce((a, [, c]) => a + c, 0);

  const pipelineSteps = [
    { label: t("policyLifecycle.submitted"), count: data?.approval_pipeline?.submitted ?? 0, color: "bg-blue-600" },
    { label: t("policyLifecycle.reviewing"), count: data?.approval_pipeline?.reviewing ?? 0, color: "bg-yellow-600" },
    { label: t("policyLifecycle.approved"), count: data?.approval_pipeline?.approved ?? 0, color: "bg-green-600" },
    { label: t("policyLifecycle.active"), count: data?.approval_pipeline?.active ?? 0, color: "bg-emerald-600" },
  ];

  const maxAge = Math.max(...(data?.policy_age_histogram ?? [{ count: 1 }]).map((h) => h.count), 1);

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">{t("policyLifecycle.title")}</h1>
          <p className="text-sm text-gray-400 mt-1">{t("policyLifecycle.subtitle")}</p>
        </div>
        <button
          onClick={refresh}
          className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition"
        >
          {t("policyLifecycle.refresh")}
        </button>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6 mb-6">
        {/* Donut: Policies by Status */}
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-sm font-semibold flex items-center gap-2 mb-4">
            <PieChart className="w-4 h-4" />
            {t("policyLifecycle.policiesByStatus")}
          </h2>
          <div className="flex items-center justify-center">
            <svg viewBox="0 0 100 100" className="w-40 h-40 -rotate-90">
              {(() => {
                let offset = 0;
                return statusEntries.map(([status, count]) => {
                  const pct = totalPolicies > 0 ? count / totalPolicies : 0;
                  const dash = pct * 251.2;
                  const circle = (
                    <circle
                      key={status}
                      cx="50"
                      cy="50"
                      r="40"
                      fill="none"
                      stroke={statusColors[status] ?? "#6b7280"}
                      strokeWidth="12"
                      strokeDasharray={`${dash} 251.2`}
                      strokeDashoffset={-offset}
                    />
                  );
                  offset += dash;
                  return circle;
                });
              })()}
            </svg>
          </div>
          <div className="space-y-1 mt-4">
            {statusEntries.map(([status, count]) => (
              <div key={status} className="flex items-center justify-between text-xs">
                <span className="flex items-center gap-2">
                  <span
                    className="w-3 h-3 rounded-sm"
                    style={{ backgroundColor: statusColors[status] ?? "#6b7280" }}
                  />
                  <span className="capitalize text-gray-300">{status}</span>
                </span>
                <span className="font-medium">{count}</span>
              </div>
            ))}
          </div>
        </div>

        {/* Approval Pipeline */}
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-sm font-semibold mb-4">{t("policyLifecycle.approvalPipeline")}</h2>
          <div className="space-y-3">
            {pipelineSteps.map((step, i) => (
              <div key={i} className="flex items-center gap-3">
                <div className={`w-10 h-10 rounded-lg ${step.color} flex items-center justify-center text-sm font-bold`}>
                  {step.count}
                </div>
                <span className="text-sm text-gray-300">{step.label}</span>
                {i < pipelineSteps.length - 1 && (
                  <div className="flex-1 h-px bg-gray-700" />
                )}
              </div>
            ))}
          </div>
          <div className="mt-4 pt-4 border-t border-gray-800">
            <div className="flex items-center gap-2 mb-1 text-gray-400">
              <Clock className="w-3 h-3" />
              <span className="text-xs">{t("policyLifecycle.avgApprovalTime")}</span>
            </div>
            <p className="text-lg font-bold">{data?.avg_approval_time_hours ?? 0}h</p>
          </div>
        </div>

        {/* Quick Stats */}
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-sm font-semibold mb-4">{t("policyLifecycle.quickStats")}</h2>
          <div className="space-y-3">
            <div className="flex items-center justify-between bg-gray-800 rounded-lg p-3">
              <span className="text-sm text-gray-400">{t("policyLifecycle.totalPolicies")}</span>
              <span className="text-lg font-bold">{totalPolicies}</span>
            </div>
            <div className="flex items-center justify-between bg-gray-800 rounded-lg p-3">
              <span className="flex items-center gap-2 text-sm text-gray-400">
                <RotateCcw className="w-3 h-3" />
                {t("policyLifecycle.rollbacks30d")}
              </span>
              <span className="text-lg font-bold text-red-400">{data?.rollback_count ?? 0}</span>
            </div>
            <div className="flex items-center justify-between bg-gray-800 rounded-lg p-3">
              <span className="flex items-center gap-2 text-sm text-gray-400">
                <GitCommit className="w-3 h-3" />
                {t("policyLifecycle.changes7d")}
              </span>
              <span className="text-lg font-bold text-blue-400">{data?.recent_changes?.length ?? 0}</span>
            </div>
          </div>
        </div>
      </div>

      {/* Recent Changes + Age Histogram */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-lg font-semibold mb-4">{t("policyLifecycle.recentChanges")}</h2>
          <div className="space-y-2">
            {(data?.recent_changes ?? []).map((change, i) => (
              <div key={i} className="flex items-center gap-3 bg-gray-800 rounded-lg p-3">
                <GitCommit className="w-4 h-4 text-blue-400 flex-shrink-0" />
                <div className="flex-1 min-w-0">
                  <p className="text-sm font-medium truncate">{change.policy_name}</p>
                  <p className="text-xs text-gray-400">
                    {change.action} by {change.author} - {change.timestamp}
                  </p>
                </div>
              </div>
            ))}
          </div>
        </div>

        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-lg font-semibold mb-4">{t("policyLifecycle.policyAgeDistribution")}</h2>
          <div className="flex items-end gap-2 h-40">
            {(data?.policy_age_histogram ?? []).map((bin, i) => (
              <div key={i} className="flex-1 flex flex-col items-center gap-1">
                <div
                  className="w-full rounded-t bg-indigo-500 hover:bg-indigo-400 transition-all"
                  style={{ height: `${(bin.count / maxAge) * 100}%`, minHeight: "4px" }}
                  title={`${bin.range}: ${bin.count} policies`}
                />
                <span className="text-xs text-gray-500 text-center whitespace-nowrap">{bin.range}</span>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}
