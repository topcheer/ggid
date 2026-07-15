"use client";

import { useFraudDetection } from "@ggid/sdk-react";
import { useTranslations } from "@/lib/i18n";
import { AlertTriangle, Activity, Fingerprint, Ban, BarChart3 } from "lucide-react";

export default function FraudDetectionPage() {
  const t = useTranslations();
  const { data, loading, error, refresh } = useFraudDetection();

  if (loading) return <div className="p-8 text-gray-400">{t("fraudDetect.loading")}</div>;
  if (error) return <div className="p-8 text-red-400">{t("common.error")}: {error}</div>;

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">{t("fraudDetect.title")}</h1>
          <p className="text-sm text-gray-400 mt-1">{t("fraudDetect.subtitle")}</p>
        </div>
        <button onClick={refresh} className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition">{t("fraudDetect.refresh")}</button>
      </div>

      {/* Summary */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
        <div className="bg-gray-900 rounded-xl p-4">
          <AlertTriangle className="w-5 h-5 text-red-400 mb-1" />
          <p className="text-xs text-gray-400">{t("fraudDetect.flaggedAccounts")}</p>
          <p className="text-xl font-bold text-red-400">{data?.flagged_accounts?.length ?? 0}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <Activity className="w-5 h-5 text-yellow-400 mb-1" />
          <p className="text-xs text-gray-400">{t("fraudDetect.velocityRules")}</p>
          <p className="text-xl font-bold">{data?.velocity_rules?.length ?? 0}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <Fingerprint className="w-5 h-5 text-blue-400 mb-1" />
          <p className="text-xs text-gray-400">{t("fraudDetect.deviceFingerprints")}</p>
          <p className="text-xl font-bold">{data?.device_fingerprint_count ?? 0}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <Ban className="w-5 h-5 text-gray-400 mb-1" />
          <p className="text-xs text-gray-400">{t("fraudDetect.blockedEntities")}</p>
          <p className="text-xl font-bold">{data?.blocked_entities.total ?? 0}</p>
        </div>
      </div>

      {/* Fraud Score Distribution */}
      <div className="bg-gray-900 rounded-xl p-6 mb-6">
        <h2 className="text-sm font-semibold flex items-center gap-2 mb-4">
          <BarChart3 className="w-4 h-4 text-purple-400" />
          {t("fraudDetect.fraudScoreDistribution")}
        </h2>
        <div className="flex items-end gap-1 h-32">
          {(data?.score_distribution ?? []).map((bucket, i) => {
            const max = Math.max(...(data?.score_distribution ?? [1]));
            const h = max > 0 ? (bucket / max) * 100 : 0;
            const color = i >= 7 ? "bg-red-500" : i >= 4 ? "bg-yellow-500" : "bg-green-500";
            return (
              <div key={i} className="flex-1 flex flex-col items-center">
                <div className={"w-full rounded-t " + color} style={{ height: h + "%" }} />
                <span className="text-xs text-gray-500 mt-1">{i * 10}-{i * 10 + 10}</span>
              </div>
            );
          })}
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Flagged Accounts */}
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-lg font-semibold mb-4">{t("fraudDetect.flaggedAccounts")}</h2>
          <div className="space-y-2 max-h-80 overflow-y-auto">
            {(data?.flagged_accounts ?? []).map((a) => (
              <div key={a.user} className="bg-gray-800 rounded-lg p-3">
                <div className="flex items-start justify-between mb-1">
                  <p className="text-sm font-medium">{a.user}</p>
                  <span className={"text-lg font-bold " + (a.score > 70 ? "text-red-400" : a.score > 40 ? "text-yellow-400" : "text-green-400")}>
                    {a.score}
                  </span>
                </div>
                <div className="flex flex-wrap gap-1 mb-2">
                  {a.signals.map((s) => (
                    <span key={s} className="text-xs px-1.5 py-0.5 bg-gray-700 rounded">{s}</span>
                  ))}
                </div>
                <span className={"text-xs px-2 py-0.5 rounded " + (
                  a.action_taken === "blocked" ? "bg-red-900 text-red-300" :
                  a.action_taken === "challenged" ? "bg-yellow-900 text-yellow-300" :
                  "bg-blue-900 text-blue-300"
                )}>
                  {a.action_taken}
                </span>
              </div>
            ))}
          </div>
        </div>

        {/* Velocity Rules + Blocked Entities */}
        <div className="space-y-6">
          <div className="bg-gray-900 rounded-xl p-6">
            <h2 className="text-sm font-semibold mb-4">{t("fraudDetect.velocityRules")}</h2>
            <div className="space-y-2">
              {(data?.velocity_rules ?? []).map((r) => (
                <div key={r.rule} className="bg-gray-800 rounded-lg p-3">
                  <div className="flex items-center justify-between mb-1">
                    <p className="text-sm font-medium">{r.rule}</p>
                    <span className={"text-xs " + (r.triggered_count > 0 ? "text-red-400" : "text-green-400")}>
                      {t("fraudDetect.triggers").replace("{count}", String(r.triggered_count))}
                    </span>
                  </div>
                  <div className="flex items-center gap-2">
                    <div className="flex-1 h-1.5 bg-gray-700 rounded-full">
                      <div className={"h-full rounded-full " + (r.current_rate / r.threshold > 0.8 ? "bg-red-500" : "bg-green-500")} style={{ width: Math.min((r.current_rate / r.threshold) * 100, 100) + "%" }} />
                    </div>
                    <span className="text-xs text-gray-400">{r.current_rate}/{r.threshold}</span>
                  </div>
                </div>
              ))}
            </div>
          </div>

          <div className="bg-gray-900 rounded-xl p-6">
            <h2 className="text-sm font-semibold flex items-center gap-2 mb-4">
              <Ban className="w-4 h-4 text-red-400" />
              {t("fraudDetect.blockedEntities")}
            </h2>
            <div className="grid grid-cols-3 gap-3">
              <div className="bg-gray-800 rounded-lg p-3 text-center">
                <p className="text-xs text-gray-400">{t("fraudDetect.ips")}</p>
                <p className="text-lg font-bold">{data?.blocked_entities.ips ?? 0}</p>
              </div>
              <div className="bg-gray-800 rounded-lg p-3 text-center">
                <p className="text-xs text-gray-400">{t("fraudDetect.emails")}</p>
                <p className="text-lg font-bold">{data?.blocked_entities.emails ?? 0}</p>
              </div>
              <div className="bg-gray-800 rounded-lg p-3 text-center">
                <p className="text-xs text-gray-400">{t("fraudDetect.devices")}</p>
                <p className="text-lg font-bold">{data?.blocked_entities.devices ?? 0}</p>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
