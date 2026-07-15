"use client";

import { useSecurityPostureScoring } from "@ggid/sdk-react";
import { useTranslations } from "@/lib/i18n";
import { Shield, TrendingUp, Award, AlertTriangle } from "lucide-react";

export default function SecurityPostureScoringPage() {
  const t = useTranslations();
  const { data, loading, error, refresh } = useSecurityPostureScoring();

  if (loading) return <div className="p-8 text-gray-400">{t("securityPosture.loading")}</div>;
  if (error) return <div className="p-8 text-red-400">{t("common.error")}: {error}</div>;

  const catColors: Record<string, string> = {
    identity: "#3b82f6",
    access: "#22c55e",
    data: "#a855f7",
    infra: "#eab308",
    compliance: "#ef4444",
  };

  const scoreChange = (data?.trend_30d?.[(data?.trend_30d?.length ?? 1) - 1] ?? 0) - (data?.trend_30d?.[0] ?? 0);

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">{t("securityPosture.title")}</h1>
          <p className="text-sm text-gray-400 mt-1">{t("securityPosture.subtitle")}</p>
        </div>
        <button onClick={refresh} className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition">{t("securityPosture.refresh")}</button>
      </div>

      {/* Overall Score Gauge + Benchmark */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6 mb-6">
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-sm font-semibold mb-4">{t("securityPosture.overallScore")}</h2>
          <div className="relative w-32 h-32 mx-auto">
            <svg className="w-32 h-32 -rotate-90" viewBox="0 0 100 100">
              <circle cx="50" cy="50" r="40" fill="none" stroke="#374151" strokeWidth="12" />
              <circle cx="50" cy="50" r="40" fill="none" stroke={data?.overall_score ?? 0 >= 80 ? "#22c55e" : "#eab308"} strokeWidth="12" strokeDasharray={((data?.overall_score ?? 0) / 100 * 251.2) + " " + 251.2} strokeLinecap="round" />
            </svg>
            <div className="absolute inset-0 flex items-center justify-center">
              <span className="text-3xl font-bold">{data?.overall_score ?? 0}</span>
            </div>
          </div>
        </div>

        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-sm font-semibold flex items-center gap-2 mb-4">
            <Award className="w-4 h-4 text-purple-400" />
            {t("securityPosture.benchmarkComparison")}
          </h2>
          <div className="space-y-3">
            <div className="flex items-center justify-between">
              <span className="text-sm">{t("securityPosture.yourScore")}</span>
              <span className="text-xl font-bold text-blue-400">{data?.overall_score ?? 0}</span>
            </div>
            <div className="flex items-center justify-between">
              <span className="text-sm text-gray-400">{t("securityPosture.industryAvg")}</span>
              <span className="text-lg font-bold text-gray-400">{data?.benchmark_comparison?.industry_avg ?? 0}</span>
            </div>
            <div className="flex items-center justify-between">
              <span className="text-sm text-gray-400">{t("securityPosture.top10Pct")}</span>
              <span className="text-lg font-bold text-green-400">{data?.benchmark_comparison?.top_10_pct ?? 0}</span>
            </div>
          </div>
        </div>

        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-sm font-semibold flex items-center gap-2 mb-4">
            <TrendingUp className="w-4 h-4 text-green-400" />
            {t("securityPosture.trend30d")}
          </h2>
          <div className="flex items-end gap-1 h-24">
            {(data?.trend_30d ?? []).map((v, i) => {
              const max = Math.max(...(data?.trend_30d ?? [1]));
              return <div key={i} className="flex-1 bg-blue-500 rounded-t" style={{ height: max > 0 ? (v / max) * 100 + "%" : "0" }} />;
            })}
          </div>
          <p className="text-xs text-gray-500 mt-2">{t("securityPosture.scoreChange").replace("{value}", String(scoreChange >= 0 ? "+" + scoreChange : scoreChange))} {t("securityPosture.points")}</p>
        </div>
      </div>

      {/* By Category Scores */}
      <div className="bg-gray-900 rounded-xl p-6 mb-6">
        <h2 className="text-sm font-semibold mb-4">{t("securityPosture.scoresByCategory")}</h2>
        <div className="space-y-3">
          {(data?.by_category ?? []).map((cat) => (
            <div key={cat.category} className="flex items-center gap-3">
              <span className="text-sm w-24 capitalize">{cat.category}</span>
              <div className="flex-1 h-3 bg-gray-800 rounded-full">
                <div className="h-full rounded-full" style={{ width: cat.score + "%", backgroundColor: catColors[cat.category] ?? "#6b7280" }} />
              </div>
              <span className="text-sm font-bold w-8 text-right">{cat.score}</span>
              <span className={"text-xs w-12 text-right " + (cat.delta > 0 ? "text-green-400" : "text-red-400")}>
                {cat.delta > 0 ? "+" : ""}{cat.delta}
              </span>
            </div>
          ))}
        </div>
      </div>

      {/* Improvement Recommendations */}
      <div className="bg-gray-900 rounded-xl p-6">
        <h2 className="text-sm font-semibold flex items-center gap-2 mb-4">
          <AlertTriangle className="w-4 h-4 text-yellow-400" />
          {t("securityPosture.improvementRecommendations")}
        </h2>
        <div className="space-y-2">
          {(data?.improvement_recommendations ?? []).map((r, i) => (
            <div key={i} className="flex items-start gap-3 bg-gray-800 rounded-lg p-3">
              <span className="text-xs font-bold text-blue-400 mt-0.5">{i + 1}.</span>
              <div className="flex-1">
                <p className="text-sm font-medium">{r.recommendation}</p>
                <p className="text-xs text-gray-400">{t("securityPosture.recommendationMeta").replace("{category}", r.category).replace("{gain}", String(r.potential_gain))}</p>
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
