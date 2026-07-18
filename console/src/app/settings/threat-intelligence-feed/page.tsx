"use client";

import { useThreatIntelligenceFeed } from "@ggid/sdk-react";
import { useTranslations } from "@/lib/i18n";
import { Radar, Database, Activity, Shield, Zap, Globe } from "lucide-react";

export default function ThreatIntelligenceFeedPage() {
  const t = useTranslations();
  const { data, loading, error, refresh } = useThreatIntelligenceFeed();

  if (loading) return <div className="p-8 text-gray-400">{t("threatIntel.loading")}</div>;
  if (error) return <div className="p-8 text-red-400">{t("common.error")}: {error}</div>;

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">{t("threatIntel.title")}</h1>
          <p className="text-sm text-gray-400 mt-1">{t("threatIntel.subtitle")}</p>
        </div>
        <button aria-label="action" onClick={refresh} className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition">{t("threatIntel.refresh")}</button>
      </div>

      {/* Summary */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
        <div className="bg-gray-900 rounded-xl p-4">
          <Database className="w-5 h-5 text-blue-400 mb-1" />
          <p className="text-xs text-gray-400">{t("threatIntel.intelSources")}</p>
          <p className="text-xl font-bold">{data?.intel_sources?.length ?? 0}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <Radar className="w-5 h-5 text-purple-400 mb-1" />
          <p className="text-xs text-gray-400">{t("threatIntel.totalIndicators")}</p>
          <p className="text-xl font-bold">{data?.indicators?.length ?? 0}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <Shield className="w-5 h-5 text-green-400 mb-1" />
          <p className="text-xs text-gray-400">{t("threatIntel.autoBlockRules")}</p>
          <p className="text-xl font-bold">{data?.auto_block_rules?.filter((r) => r.enabled).length ?? 0}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <Activity className="w-5 h-5 text-yellow-400 mb-1" />
          <p className="text-xs text-gray-400">{t("threatIntel.avgConfidence")}</p>
          <p className="text-xl font-bold">
            {data?.indicators?.length ? Math.round(data.indicators.reduce((a, i) => a + i.confidence, 0) / data.indicators.length * 100) : 0}%
          </p>
        </div>
      </div>

      {/* Intel Sources + Feed Health */}
      <div className="bg-gray-900 rounded-xl p-6 mb-6">
        <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
          <Database className="w-5 h-5 text-blue-400" />
          {t("threatIntel.intelligenceSources")}
        </h2>
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3">
          {(data?.intel_sources ?? []).map((src) => {
            const health = data?.feed_health?.find((h) => h.source_name === src.source_name);
            return (
              <div key={src.source_name} className="bg-gray-800 rounded-lg p-3">
                <div className="flex items-center justify-between mb-2">
                  <p className="text-sm font-semibold">{src.source_name}</p>
                  <span className={"text-xs px-2 py-0.5 rounded " + (src.status === "active" ? "bg-green-900 text-green-300" : "bg-red-900 text-red-300")}>
                    {src.status}
                  </span>
                </div>
                <p className="text-xs text-gray-400">{t("threatIntel.type")} {src.type}</p>
                <p className="text-xs text-gray-500">{t("threatIntel.lastSync")} {src.last_sync}</p>
                {health && (
                  <div className="mt-2 flex items-center gap-2">
                    <div className="flex-1 h-1 bg-gray-700 rounded-full">
                      <div className="h-full bg-blue-500 rounded-full" style={{ width: (health.uptime_pct) + "%" }} />
                    </div>
                    <span className="text-xs text-gray-400">{health.uptime_pct}% {t("threatIntel.uptime")}</span>
                  </div>
                )}
              </div>
            );
          })}
        </div>
      </div>

      {/* Indicators Table */}
      <div className="bg-gray-900 rounded-xl p-6 mb-6">
        <h2 className="text-sm font-semibold mb-4">{t("threatIntel.threatIndicators")}</h2>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-gray-800 text-gray-400">
                <th scope="col" className="text-left py-2 pr-3">{t("threatIntel.indicator")}</th>
                <th scope="col" className="text-left py-2 pr-3">{t("threatIntel.type")}</th>
                <th scope="col" className="text-left py-2 pr-3">{t("threatIntel.confidence")}</th>
                <th scope="col" className="text-left py-2 pr-3">{t("threatIntel.firstSeen")}</th>
                <th scope="col" className="text-left py-2 pr-3">{t("threatIntel.lastSeen")}</th>
                <th scope="col" className="text-left py-2 pr-3">{t("threatIntel.source")}</th>
                <th scope="col" className="text-left py-2 pr-3">{t("threatIntel.tags")}</th>
              </tr>
            </thead>
            <tbody>
              {(data?.indicators ?? []).map((ind: any, i: number) => (
                <tr key={i} className="border-b border-gray-800">
                  <td className="py-3 pr-3 font-mono text-xs text-blue-400">{ind.indicator}</td>
                  <td className="py-3 pr-3">
                    <span className="text-xs px-2 py-0.5 rounded bg-gray-800">{ind.type}</span>
                  </td>
                  <td className="py-3 pr-3">
                    <div className="flex items-center gap-2">
                      <div className="w-12 h-1.5 bg-gray-700 rounded-full">
                        <div className={"h-full rounded-full " + (ind.confidence > 0.8 ? "bg-red-500" : ind.confidence > 0.5 ? "bg-yellow-500" : "bg-green-500")} style={{ width: (ind.confidence * 100) + "%" }} />
                      </div>
                      <span className="text-xs">{(ind.confidence * 100).toFixed(0)}%</span>
                    </div>
                  </td>
                  <td className="py-3 pr-3 text-xs text-gray-400">{ind.first_seen}</td>
                  <td className="py-3 pr-3 text-xs text-gray-400">{ind.last_seen}</td>
                  <td className="py-3 pr-3 text-xs text-gray-400">{ind.source}</td>
                  <td className="py-3 pr-3">
                    <div className="flex flex-wrap gap-1">
                      {ind.tags.map((tag) => (
                        <span key={tag} className="text-xs px-1.5 py-0.5 bg-purple-900/50 text-purple-300 rounded">{tag}</span>
                      ))}
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      {/* Auto-Block Rules */}
      <div className="bg-gray-900 rounded-xl p-6">
        <h2 className="text-sm font-semibold flex items-center gap-2 mb-4">
          <Zap className="w-4 h-4 text-yellow-400" />
          {t("threatIntel.autoBlockRules")}
        </h2>
        <div className="space-y-2">
          {(data?.auto_block_rules ?? []).map((rule) => (
            <div key={rule.rule_name} className="flex items-center justify-between bg-gray-800 rounded-lg p-3">
              <div>
                <p className="text-sm font-medium">{rule.rule_name}</p>
                <p className="text-xs text-gray-400">{rule.description}</p>
              </div>
              <span className={"text-xs px-2 py-0.5 rounded " + (rule.enabled ? "bg-green-900 text-green-300" : "bg-gray-700 text-gray-400")}>
                {rule.enabled ? t("common.enabled") : t("common.disabled")}
              </span>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
