"use client";

import { useIdentityRoleMiningConfig } from "@ggid/sdk-react";
import { Pickaxe, Brain, CheckCircle, Clock, BarChart3 } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

export default function IdentityRoleMiningConfigPage() {
  const { data, loading, error, refresh, runMining } = useIdentityRoleMiningConfig();
  const t = useTranslations();

  if (loading) return <div className="p-8 text-gray-400">{t("idRoleMiningConfig.loading")}</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">{t("idRoleMiningConfig.title")}</h1>
          <p className="text-sm text-gray-400 mt-1">{t("idRoleMiningConfig.subtitle")}</p>
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={() => runMining()}
            className="flex items-center gap-2 px-4 py-2 bg-purple-600 hover:bg-purple-700 rounded-lg text-sm font-medium transition"
          >
            <Pickaxe className="w-4 h-4" />
            Run Mining
          </button>
          <button
            onClick={refresh}
            className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition"
          >
            Refresh
          </button>
        </div>
      </div>

      {/* Mining Parameters */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-blue-400">
            <BarChart3 className="w-4 h-4" />
            <span className="text-xs text-gray-400">{t("idRoleMiningConfig.minUsage")}</span>
          </div>
          <p className="text-xl font-bold">{data?.mining_parameters.min_usage_threshold ?? 0}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-purple-400">
            <Clock className="w-4 h-4" />
            <span className="text-xs text-gray-400">{t("idRoleMiningConfig.coOccurrenceWindow")}</span>
          </div>
          <p className="text-xl font-bold">{data?.mining_parameters.co_occurrence_window_days ?? 0}d</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-green-400">
            <Brain className="w-4 h-4" />
            <span className="text-xs text-gray-400">{t("idRoleMiningConfig.minConfidence")}</span>
          </div>
          <p className="text-xl font-bold">{((data?.mining_parameters.confidence_score_min ?? 0) * 100).toFixed(0)}%</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-yellow-400">
            <CheckCircle className="w-4 h-4" />
            <span className="text-xs text-gray-400">{t("idRoleMiningConfig.appliedRoles")}</span>
          </div>
          <p className="text-xl font-bold">{data?.applied_count ?? 0}</p>
        </div>
      </div>

      {/* Algorithm & Settings */}
      <div className="bg-gray-900 rounded-xl p-4 mb-6">
        <div className="flex items-center justify-between flex-wrap gap-3">
          <div className="flex items-center gap-4">
            <div>
              <span className="text-xs text-gray-400 block">{t("idRoleMiningConfig.similarityAlgorithm")}</span>
              <span className="text-sm font-medium capitalize">{data?.similarity_algorithm}</span>
            </div>
            <div>
              <span className="text-xs text-gray-400 block">{t("idRoleMiningConfig.autoSuggest")}</span>
              <span className={"text-sm font-medium " + (data?.auto_suggest_roles ? "text-green-400" : "text-red-400")}>
                {data?.auto_suggest_roles ? "Enabled" : "Disabled"}
              </span>
            </div>
            <div>
              <span className="text-xs text-gray-400 block">{t("idRoleMiningConfig.lastRun")}</span>
              <span className="text-sm font-medium">{data?.last_mining_run ?? "Never"}</span>
            </div>
          </div>
        </div>
      </div>

      {/* Suggested Roles Review Queue */}
      <div className="bg-gray-900 rounded-xl p-6">
        <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
          <Pickaxe className="w-5 h-5 text-purple-400" />
          Suggested Roles Review Queue
        </h2>
        <div className="space-y-2">
          {(data?.suggested_roles_review_queue ?? []).map((role, i) => (
            <div key={i} className="bg-gray-800 rounded-lg p-3">
              <div className="flex items-center justify-between mb-2">
                <div>
                  <p className="text-sm font-medium">{role.suggested_name}</p>
                  <p className="text-xs text-gray-400">{role.member_count} users - {role.permission_count} permissions</p>
                </div>
                <div className="flex items-center gap-3">
                  <div className="flex items-center gap-1">
                    <span className="text-xs text-gray-400">{t("idRoleMiningConfig.confidence")}</span>
                    <div className="w-16 bg-gray-700 rounded-full h-1.5">
                      <div
                        className="bg-purple-500 rounded-full h-1.5"
                        style={{ width: `${role.confidence_score * 100}%` }}
                      />
                    </div>
                    <span className="text-xs font-medium">{Math.round(role.confidence_score * 100)}%</span>
                  </div>
                  <button aria-label="action" className="text-xs px-2 py-1 bg-green-600 hover:bg-green-700 rounded">{t("idRoleMiningConfig.accept")}</button>
                  <button aria-label="action" className="text-xs px-2 py-1 bg-gray-600 hover:bg-gray-500 rounded">{t("idRoleMiningConfig.reject")}</button>
                </div>
              </div>
              <div className="flex flex-wrap gap-1">
                {role.key_permissions.slice(0, 5).map((p) => (
                  <span key={p} className="text-xs px-1.5 py-0.5 bg-gray-700 rounded text-gray-300">{p}</span>
                ))}
                {role.key_permissions.length > 5 && (
                  <span className="text-xs text-gray-500">+{role.key_permissions.length - 5} more</span>
                )}
              </div>
            </div>
          ))}
          {(data?.suggested_roles_review_queue ?? []).length === 0 && (
            <p className="text-sm text-gray-500 text-center py-4">{t("idRoleMiningConfig.noSuggestions")}</p>
          )}
        </div>
      </div>
    </div>
  );
}
