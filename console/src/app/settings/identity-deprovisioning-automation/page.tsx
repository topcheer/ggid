"use client";

import { useIdentityDeprovisioningAutomation } from "@ggid/sdk-react";
import { Bot, Play, Pause, CheckCircle, Clock, ArrowRight, Activity } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

export default function IdentityDeprovisioningAutomationPage() {
  const { data, loading, error, refresh } = useIdentityDeprovisioningAutomation();
  const t = useTranslations();

  if (loading) return <div className="p-8 text-gray-400">{t("idDeprovisionAuto.loading")}</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">{t("idDeprovisionAuto.title")}</h1>
          <p className="text-sm text-gray-400 mt-1">{t("idDeprovisionAuto.subtitle")}</p>
        </div>
        <div className="flex items-center gap-2">
          <span
            className={"flex items-center gap-1 px-3 py-2 rounded-lg text-xs font-medium " + (
              data?.dry_run ? "bg-yellow-900 text-yellow-300" : "bg-green-900 text-green-300"
            )}
          >
            {data?.dry_run ? <Play className="w-3 h-3" /> : <Pause className="w-3 h-3" />}
            {data?.dry_run ? "Dry Run" : "Live Mode"}
          </span>
          <button
            onClick={refresh}
            className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition"
           aria-label="Action">
            {t("idDeprovisionAuto.refresh")}
          </button>
        </div>
      </div>

      {/* Summary Stats */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-blue-400">
            <Bot className="w-4 h-4" />
            <span className="text-xs text-gray-400">{t("idDeprovisionAuto.automationRules")}</span>
          </div>
          <p className="text-2xl font-bold">{data?.automation_rules?.length ?? 0}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-yellow-400">
            <Clock className="w-4 h-4" />
            <span className="text-xs text-gray-400">{t("idDeprovisionAuto.pendingActions")}</span>
          </div>
          <p className="text-2xl font-bold">{data?.pending_actions?.length ?? 0}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-green-400">
            <CheckCircle className="w-4 h-4" />
            <span className="text-xs text-gray-400">{t("idDeprovisionAuto.successRate")}</span>
          </div>
          <p className="text-2xl font-bold text-green-400">{data?.success_rate_pct ?? 0}%</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-purple-400">
            <Activity className="w-4 h-4" />
            <span className="text-xs text-gray-400">{t("idDeprovisionAuto.processed7d")}</span>
          </div>
          <p className="text-2xl font-bold">{data?.processed_7d ?? 0}</p>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Automation Rules */}
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-lg font-semibold mb-4">{t("idDeprovisionAuto.automationRules")}</h2>
          <div className="space-y-2">
            {(data?.automation_rules ?? []).map((rule: any, i: number) => (
              <div key={i} className="bg-gray-800 rounded-lg p-3">
                <div className="flex items-center justify-between mb-2">
                  <div>
                    <p className="text-sm font-medium capitalize">{rule.trigger.replace(/_/g, " ")}</p>
                    <p className="text-xs text-gray-400">{t("idDeprovisionAuto.action")} {rule.action} - {t("idDeprovisionAuto.delay")} {rule.delay_hours}h</p>
                  </div>
                  <span
                    className={"text-xs px-2 py-0.5 rounded " + (
                      rule.enabled ? "bg-green-900 text-green-300" : "bg-gray-700 text-gray-400"
                    )}
                  >
                    {rule.enabled ? "Active" : "Disabled"}
                  </span>
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* Pending Actions Queue */}
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
            <Clock className="w-5 h-5 text-yellow-400" />
            {t("idDeprovisionAuto.pendingActions")}
          </h2>
          <div className="space-y-2 max-h-64 overflow-y-auto">
            {(data?.pending_actions ?? []).map((action: any, i: number) => (
              <div key={i} className="bg-gray-800 rounded-lg p-3">
                <div className="flex items-center justify-between mb-1">
                  <p className="text-sm font-medium">{action.user}</p>
                  <span className="text-xs text-gray-400">{action.action}</span>
                </div>
                <p className="text-xs text-gray-400">{t("idDeprovisionAuto.trigger")} {action.trigger_reason}</p>
                <p className="text-xs text-gray-500">{t("idDeprovisionAuto.scheduled")} {action.scheduled_at}</p>
              </div>
            ))}
            {(data?.pending_actions ?? []).length === 0 && (
              <p className="text-sm text-gray-500 text-center py-4">{t("idDeprovisionAuto.noPending")}</p>
            )}
          </div>
        </div>
      </div>

      {/* Workflow Visual */}
      <div className="bg-gray-900 rounded-xl p-6 mt-6">
        <h2 className="text-lg font-semibold mb-4">{t("idDeprovisionAuto.workflow")}</h2>
        <div className="flex items-center gap-2 flex-wrap">
          {[
            "1. Trigger (HR/Policy/Manual)",
            "2. Approval (Manager)",
            "3. Execute (Disable + Revoke)",
            "4. Verify (Access Check)",
          ].map((step: any, i: number) => (
            <div key={i} className="flex items-center gap-2">
              <span className="text-xs px-3 py-1.5 bg-gray-800 rounded-lg border border-gray-700">{step}</span>
              {i < 3 && <ArrowRight className="w-3 h-3 text-gray-500" />}
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
