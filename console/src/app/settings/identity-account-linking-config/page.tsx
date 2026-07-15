"use client";

import { useIdentityAccountLinkingConfig } from "@ggid/sdk-react";
import { Link2, Unlink, ShieldCheck, GitMerge, Users, AlertTriangle } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

export default function IdentityAccountLinkingConfigPage() {
  const { data, loading, error, refresh } = useIdentityAccountLinkingConfig();
  const t = useTranslations();

  if (loading) return <div className="p-8 text-gray-400">{t("idAccountLinking.loading")}</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">{t("idAccountLinking.title")}</h1>
          <p className="text-sm text-gray-400 mt-1">{t("idAccountLinking.subtitle")}</p>
        </div>
        <button
          onClick={refresh}
          className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition"
        >
          Refresh
        </button>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-blue-400">
            <Link2 className="w-4 h-4" />
            <span className="text-xs text-gray-400">{t("idAccountLinking.linkedAccounts")}</span>
          </div>
          <p className="text-2xl font-bold">{(data?.linked_accounts_stats?.total_linked ?? 0).toLocaleString()}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-green-400">
            <Users className="w-4 h-4" />
            <span className="text-xs text-gray-400">{t("idAccountLinking.autoLinked24h")}</span>
          </div>
          <p className="text-2xl font-bold">{data?.linked_accounts_stats?.auto_linked_24h ?? 0}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-yellow-400">
            <AlertTriangle className="w-4 h-4" />
            <span className="text-xs text-gray-400">{t("idAccountLinking.conflicts24h")}</span>
          </div>
          <p className="text-2xl font-bold">{data?.linked_accounts_stats?.conflicts_24h ?? 0}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-red-400">
            <Unlink className="w-4 h-4" />
            <span className="text-xs text-gray-400">{t("idAccountLinking.unlinked24h")}</span>
          </div>
          <p className="text-2xl font-bold">{data?.linked_accounts_stats?.unlinked_24h ?? 0}</p>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Linking Methods */}
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
            <Link2 className="w-5 h-5 text-blue-400" />
            Linking Methods
          </h2>
          <div className="space-y-2">
            {(data?.linking_methods ?? []).map((m) => (
              <div key={m.method} className="flex items-center justify-between bg-gray-800 rounded-lg p-3">
                <div>
                  <p className="text-sm font-medium capitalize">{m.method.replace(/_/g, " ")}</p>
                  <p className="text-xs text-gray-400">{m.description}</p>
                </div>
                <span
                  className={"text-xs px-2 py-0.5 rounded " + (
                    m.enabled ? "bg-green-900 text-green-300" : "bg-gray-700 text-gray-400"
                  )}
                >
                  {m.enabled ? "Enabled" : "Disabled"}
                </span>
              </div>
            ))}
          </div>

          <div className="mt-4 pt-4 border-t border-gray-800">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <ShieldCheck className="w-4 h-4 text-green-400" />
                <span className="text-sm text-gray-300">{t("idAccountLinking.requireVerification")}</span>
              </div>
              <span
                className={"text-xs px-2 py-0.5 rounded " + (
                  data?.require_verification ? "bg-green-900 text-green-300" : "bg-gray-700 text-gray-400"
                )}
              >
                {data?.require_verification ? "Required" : "Optional"}
              </span>
            </div>
          </div>

          <div className="mt-3">
            <div className="flex items-center justify-between bg-gray-800 rounded-lg p-3">
              <span className="text-sm text-gray-300">{t("idAccountLinking.autoLinkThreshold")}</span>
              <span className="text-sm font-bold text-blue-400">{Math.round((data?.auto_link_threshold ?? 0) * 100)}%</span>
            </div>
          </div>
        </div>

        <div className="space-y-6">
          {/* Conflict Resolution */}
          <div className="bg-gray-900 rounded-xl p-6">
            <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
              <GitMerge className="w-5 h-5 text-purple-400" />
              Conflict Resolution
            </h2>
            <div className="space-y-2">
              <div className="flex items-center justify-between bg-gray-800 rounded-lg p-3">
                <span className="text-sm text-gray-300">{t("idAccountLinking.strategy")}</span>
                <span className="text-sm font-medium capitalize">{data?.conflict_resolution?.strategy?.replace(/_/g, " ") ?? "manual"}</span>
              </div>
              <p className="text-xs text-gray-400">{data?.conflict_resolution?.description ?? ""}</p>
            </div>
          </div>

          {/* Unlink Policy */}
          <div className="bg-gray-900 rounded-xl p-6">
            <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
              <Unlink className="w-5 h-5 text-red-400" />
              Unlink Policy
            </h2>
            <div className="space-y-2">
              <div className="bg-gray-800 rounded-lg p-3">
                <div className="flex items-center justify-between mb-1">
                  <span className="text-sm text-gray-300">{t("idAccountLinking.allowSelfUnlink")}</span>
                  <span
                    className={"text-xs px-2 py-0.5 rounded " + (
                      data?.unlink_policy?.allow_self_service ? "bg-green-900 text-green-300" : "bg-gray-700 text-gray-400"
                    )}
                  >
                    {data?.unlink_policy?.allow_self_service ? "Yes" : "No"}
                  </span>
                </div>
              </div>
              <div className="bg-gray-800 rounded-lg p-3">
                <div className="flex items-center justify-between mb-1">
                  <span className="text-sm text-gray-300">{t("idAccountLinking.gracePeriod")}</span>
                  <span className="text-sm font-medium">{data?.unlink_policy?.grace_period_hours ?? 0}h</span>
                </div>
              </div>
              <div className="bg-gray-800 rounded-lg p-3">
                <div className="flex items-center justify-between mb-1">
                  <span className="text-sm text-gray-300">{t("idAccountLinking.adminApproval")}</span>
                  <span
                    className={"text-xs px-2 py-0.5 rounded " + (
                      data?.unlink_policy?.require_admin_approval ? "bg-yellow-900 text-yellow-300" : "bg-gray-700 text-gray-400"
                    )}
                  >
                    {data?.unlink_policy?.require_admin_approval ? "Yes" : "No"}
                  </span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
