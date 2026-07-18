"use client";

import { useIdentityJitProvisioningConfig } from "@ggid/sdk-react";
import { Zap, ArrowRight, Users, Activity, Clock } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

export default function IdentityJitProvisioningConfigPage() {
  const { data, loading, error, refresh } = useIdentityJitProvisioningConfig();
  const t = useTranslations();

  if (loading) return <div className="p-8 text-gray-400">{t("idJitProvisioning.loading")}</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">{t("idJitProvisioning.title")}</h1>
          <p className="text-sm text-gray-400 mt-1">{t("idJitProvisioning.subtitle")}</p>
        </div>
        <button
          onClick={refresh}
          className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition"
        >
          Refresh
        </button>
      </div>

      {/* Summary */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-green-400">
            <Zap className="w-4 h-4" />
            <span className="text-xs text-gray-400">{t("idJitProvisioning.jitEnabled")}</span>
          </div>
          <p className="text-lg font-bold">{data?.per_idp_config?.some((p) => p.enabled) ? "Yes" : "No"}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-blue-400">
            <Users className="w-4 h-4" />
            <span className="text-xs text-gray-400">{t("idJitProvisioning.defaultRole")}</span>
          </div>
          <p className="text-sm font-bold">{data?.default_role_on_create ?? "user"}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-purple-400">
            <Activity className="w-4 h-4" />
            <span className="text-xs text-gray-400">{t("idJitProvisioning.updateOnLogin")}</span>
          </div>
          <p className="text-sm font-bold">{data?.update_on_login ? "Yes" : "No"}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-yellow-400">
            <Clock className="w-4 h-4" />
            <span className="text-xs text-gray-400">{t("idJitProvisioning.conflictResolution")}</span>
          </div>
          <p className="text-sm font-bold capitalize">{data?.conflict_resolution ?? "create"}</p>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Per-IdP Config */}
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-sm font-semibold mb-4">{t("idJitProvisioning.perIdp")}</h2>
          <div className="space-y-2">
            {(data?.per_idp_config ?? []).map((p) => (
              <div key={p.idp_name} className="flex items-center justify-between bg-gray-800 rounded-lg p-3">
                <div>
                  <p className="text-sm font-medium">{p.idp_name}</p>
                  <p className="text-xs text-gray-400">{p.enabled ? "Auto-creates users on login" : "Disabled"}</p>
                </div>
                <span className={"text-xs px-2 py-0.5 rounded " + (p.enabled ? "bg-green-900 text-green-300" : "bg-gray-700 text-gray-400")}>
                  {p.enabled ? "Enabled" : "Disabled"}
                </span>
              </div>
            ))}
          </div>
        </div>

        {/* Default Groups */}
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-sm font-semibold mb-4">{t("idJitProvisioning.defaultGroups")}</h2>
          <div className="flex flex-wrap gap-2">
            {(data?.default_group_assignments ?? []).map((g) => (
              <span key={g} className="text-xs px-3 py-1.5 bg-gray-800 rounded-lg border border-gray-700">{g}</span>
            ))}
          </div>
        </div>
      </div>

      {/* Attribute Mapping Table */}
      <div className="bg-gray-900 rounded-xl p-6 mt-6">
        <h2 className="text-lg font-semibold mb-4">{t("idJitProvisioning.attributeMapping")}</h2>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-gray-800 text-gray-400">
                <th scope="col" className="text-left py-2 pr-3">{t("idJitProvisioning.idpClaim")}</th>
                <th scope="col" className="text-left py-2 pr-3"></th>
                <th scope="col" className="text-left py-2 pr-3">{t("idJitProvisioning.localAttribute")}</th>
                <th scope="col" className="text-left py-2 pr-3">{t("idJitProvisioning.required")}</th>
              </tr>
            </thead>
            <tbody>
              {(data?.attribute_mapping ?? []).map((m: any, i: number) => (
                <tr key={i} className="border-b border-gray-800">
                  <td className="py-3 pr-3 font-mono text-xs text-blue-400">{m.idp_claim}</td>
                  <td className="py-3 pr-3"><ArrowRight className="w-3 h-3 text-gray-500" /></td>
                  <td className="py-3 pr-3 font-mono text-xs text-green-400">{m.local_attr}</td>
                  <td className="py-3 pr-3">
                    <span className={"text-xs " + (m.required ? "text-red-400" : "text-gray-500")}>
                      {m.required ? "Yes" : "No"}
                    </span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      {/* Provisioning Log */}
      <div className="bg-gray-900 rounded-xl p-6 mt-6">
        <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
          <Activity className="w-5 h-5 text-blue-400" />
          Provisioning Log (24h)
        </h2>
        <div className="space-y-2 max-h-48 overflow-y-auto">
          {(data?.provisioning_log_24h ?? []).map((entry: any, i: number) => (
            <div key={i} className="flex items-center gap-3 bg-gray-800 rounded-lg p-2">
              <span
                className={"w-2 h-2 rounded-full " + (
                  entry.action === "created" ? "bg-green-500" :
                  entry.action === "updated" ? "bg-blue-500" :
                  "bg-red-500"
                )}
              />
              <div className="flex-1">
                <p className="text-xs font-medium">{entry.user}</p>
                <p className="text-xs text-gray-500">{entry.idp} - {entry.action}</p>
              </div>
              <span className="text-xs text-gray-500">{entry.timestamp}</span>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
