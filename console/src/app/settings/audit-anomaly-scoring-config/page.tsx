"use client";

import { useAuditAnomalyScoringConfig } from "@ggid/sdk-react";
import { useTranslations } from "@/lib/i18n";
import { Sliders, Activity, RefreshCw, Brain, Target, TrendingUp } from "lucide-react";

export default function AuditAnomalyScoringConfigPage() {
  const t = useTranslations();
  const { data, loading, error, refresh, retrainModel } = useAuditAnomalyScoringConfig();

  if (loading) return <div className="p-8 text-gray-400">Loading anomaly scoring config...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  const thresholdColors: Record<string, string> = {
    low: "bg-blue-900 text-blue-300",
    medium: "bg-yellow-900 text-yellow-300",
    high: "bg-orange-900 text-orange-300",
    critical: "bg-red-900 text-red-300",
  };

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">Anomaly Scoring Configuration</h1>
          <p className="text-sm text-gray-400 mt-1">Configure audit anomaly detection signals, thresholds, and model tuning</p>
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={() => retrainModel()}
            className="flex items-center gap-2 px-4 py-2 bg-purple-600 hover:bg-purple-700 rounded-lg text-sm font-medium transition"
          >
            <Brain className="w-4 h-4" />
            Retrain Model
          </button>
          <button
            onClick={refresh}
            className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition"
          >
            Refresh
          </button>
        </div>
      </div>

      {/* Config Parameters */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-6">
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-blue-400">
            <Activity className="w-4 h-4" />
            <span className="text-xs text-gray-400">Baseline Window</span>
          </div>
          <p className="text-xl font-bold">{data?.baseline_window_hours ?? 0}h</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-yellow-400">
            <Sliders className="w-4 h-4" />
            <span className="text-xs text-gray-400">Sensitivity</span>
          </div>
          <p className="text-xl font-bold capitalize">{data?.sensitivity_adjustment ?? "normal"}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-green-400">
            <Target className="w-4 h-4" />
            <span className="text-xs text-gray-400">Accuracy</span>
          </div>
          <p className="text-xl font-bold">{data?.accuracy_stats?.accuracy_pct ?? 0}%</p>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Scoring Signals */}
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-lg font-semibold mb-4">Scoring Signals</h2>
          <div className="space-y-3">
            {(data?.scoring_signals ?? []).map((sig) => (
              <div key={sig.signal_name} className="bg-gray-800 rounded-lg p-3">
                <div className="flex items-center justify-between mb-2">
                  <p className="text-sm font-medium capitalize">{sig.signal_name.replace(/_/g, " ")}</p>
                  <span className="text-xs text-gray-400">Weight: {sig.weight}x</span>
                </div>
                <div className="flex items-center gap-2">
                  <span className="text-xs text-gray-400">Threshold:</span>
                  <div className="flex-1 bg-gray-700 rounded-full h-1.5">
                    <div
                      className="bg-blue-500 rounded-full h-1.5"
                      style={{ width: `${sig.threshold * 100}%` }}
                    />
                  </div>
                  <span className="text-xs font-medium">{Math.round(sig.threshold * 100)}%</span>
                </div>
              </div>
            ))}
          </div>
        </div>

        <div className="space-y-6">
          {/* Composite Threshold */}
          <div className="bg-gray-900 rounded-xl p-6">
            <h2 className="text-lg font-semibold mb-4">Composite Threshold</h2>
            <div className="space-y-2">
              {(["low", "medium", "high", "critical"] as const).map((level) => {
                const thresh = data?.composite_threshold?.[level] ?? 0;
                return (
                  <div key={level} className="flex items-center gap-3">
                    <span className={"text-xs px-2 py-0.5 rounded w-20 text-center " + thresholdColors[level]}>
                      {level}
                    </span>
                    <div className="flex-1 bg-gray-700 rounded-full h-2">
                      <div
                        className={level === "critical" ? "bg-red-500" : level === "high" ? "bg-orange-500" : level === "medium" ? "bg-yellow-500" : "bg-blue-500"}
                        style={{ width: `${thresh}%`, height: "100%", borderRadius: "9999px" }}
                      />
                    </div>
                    <span className="text-sm font-medium w-12 text-right">{thresh}</span>
                  </div>
                );
              })}
            </div>
          </div>

          {/* Accuracy Stats */}
          <div className="bg-gray-900 rounded-xl p-6">
            <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
              <TrendingUp className="w-5 h-5 text-green-400" />
              Model Accuracy
            </h2>
            <div className="space-y-2">
              <div className="flex items-center justify-between bg-gray-800 rounded-lg p-3">
                <span className="text-sm text-gray-300">Accuracy</span>
                <span className="text-sm font-bold text-green-400">{data?.accuracy_stats?.accuracy_pct ?? 0}%</span>
              </div>
              <div className="flex items-center justify-between bg-gray-800 rounded-lg p-3">
                <span className="text-sm text-gray-300">Precision</span>
                <span className="text-sm font-bold">{data?.accuracy_stats?.precision_pct ?? 0}%</span>
              </div>
              <div className="flex items-center justify-between bg-gray-800 rounded-lg p-3">
                <span className="text-sm text-gray-300">Recall</span>
                <span className="text-sm font-bold">{data?.accuracy_stats?.recall_pct ?? 0}%</span>
              </div>
              <div className="flex items-center justify-between bg-gray-800 rounded-lg p-3">
                <span className="text-sm text-gray-300">False Positive Rate</span>
                <span className="text-sm font-bold text-red-400">{data?.accuracy_stats?.false_positive_rate ?? 0}%</span>
              </div>
              <div className="flex items-center justify-between bg-gray-800 rounded-lg p-3">
                <span className="text-sm text-gray-300">Last Trained</span>
                <span className="text-sm text-gray-400">{data?.accuracy_stats?.last_trained ?? "N/A"}</span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
