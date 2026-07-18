"use client";
import { useTranslations } from "@/lib/i18n";

import { useAuthRiskEngineConfig } from "@ggid/sdk-react";
import { Zap, Activity, Brain, TrendingUp, RefreshCw } from "lucide-react";

export default function AuthRiskEngineConfigPage() {
  const t = useTranslations();
  const { data, loading, error, refresh, retrainModel } = useAuthRiskEngineConfig();

  if (loading) return <div className="p-8 text-gray-400">Loading risk engine config...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">Risk Engine Configuration</h1>
          <p className="text-sm text-gray-400 mt-1">Configure authentication risk scoring signals and model</p>
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={() => retrainModel()}
            className="flex items-center gap-2 px-4 py-2 bg-purple-600 hover:bg-purple-700 rounded-lg text-sm font-medium transition"
          >
            <Brain className="w-4 h-4" />
            Retrain Model
          </button>
          <button onClick={refresh} className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition">Refresh</button>
        </div>
      </div>

      {/* Summary */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-blue-400">
            <Zap className="w-4 h-4" />
            <span className="text-xs text-gray-400">Algorithm</span>
          </div>
          <p className="text-sm font-bold capitalize">{data?.scoring_algorithm ?? "weighted_sum"}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-green-400">
            <Activity className="w-4 h-4" />
            <span className="text-xs text-gray-400">Model Version</span>
          </div>
          <p className="text-sm font-bold font-mono">v{data?.model_version ?? "1.0"}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-purple-400">
            <RefreshCw className="w-4 h-4" />
            <span className="text-xs text-gray-400">Retrain Frequency</span>
          </div>
          <p className="text-sm font-bold capitalize">{data?.retraining_frequency ?? "weekly"}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-yellow-400">
            <TrendingUp className="w-4 h-4" />
            <span className="text-xs text-gray-400">Active Signals</span>
          </div>
          <p className="text-2xl font-bold">{data?.risk_signals?.length ?? 0}</p>
        </div>
      </div>

      {/* Risk Signals */}
      <div className="bg-gray-900 rounded-xl p-6 mb-6">
        <h2 className="text-lg font-semibold mb-4">Risk Signals</h2>
        <div className="space-y-2">
          {(data?.risk_signals ?? []).map((s) => (
            <div key={s.signal} className="flex items-center gap-4 bg-gray-800 rounded-lg p-3">
              <div className="flex-1">
                <p className="text-sm font-medium">{s.signal}</p>
                <p className="text-xs text-gray-400">Action: {s.action_per_trigger}</p>
              </div>
              <div className="w-32">
                <div className="flex items-center justify-between mb-1">
                  <span className="text-xs text-gray-400">Weight</span>
                  <span className="text-xs font-medium">{s.weight}</span>
                </div>
                <div className="h-1.5 bg-gray-700 rounded-full">
                  <div className="h-full bg-blue-500 rounded-full" style={{ width: (s.weight * 100) + "%" }} />
                </div>
              </div>
              <div className="w-24 text-right">
                <span className="text-xs text-gray-400">Threshold: </span>
                <span className="text-xs font-medium">{s.threshold}</span>
              </div>
            </div>
          ))}
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Backtest Results */}
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-sm font-semibold flex items-center gap-2 mb-4">
            <TrendingUp className="w-4 h-4 text-green-400" />
            Backtest Results
          </h2>
          <div className="space-y-3">
            <div className="bg-gray-800 rounded-lg p-3">
              <div className="flex justify-between mb-1">
                <span className="text-sm text-gray-400">Precision</span>
                <span className="text-sm font-bold text-green-400">{((data?.backtest_results?.precision ?? 0) * 100).toFixed(1)}%</span>
              </div>
              <div className="h-2 bg-gray-700 rounded-full">
                <div className="h-full bg-green-500 rounded-full" style={{ width: ((data?.backtest_results?.precision ?? 0) * 100) + "%" }} />
              </div>
            </div>
            <div className="bg-gray-800 rounded-lg p-3">
              <div className="flex justify-between mb-1">
                <span className="text-sm text-gray-400">Recall</span>
                <span className="text-sm font-bold text-blue-400">{((data?.backtest_results?.recall ?? 0) * 100).toFixed(1)}%</span>
              </div>
              <div className="h-2 bg-gray-700 rounded-full">
                <div className="h-full bg-blue-500 rounded-full" style={{ width: ((data?.backtest_results?.recall ?? 0) * 100) + "%" }} />
              </div>
            </div>
            <div className="bg-gray-800 rounded-lg p-3">
              <div className="flex justify-between mb-1">
                <span className="text-sm text-gray-400">F1 Score</span>
                <span className="text-sm font-bold text-purple-400">{((data?.backtest_results?.f1 ?? 0) * 100).toFixed(1)}%</span>
              </div>
              <div className="h-2 bg-gray-700 rounded-full">
                <div className="h-full bg-purple-500 rounded-full" style={{ width: ((data?.backtest_results?.f1 ?? 0) * 100) + "%" }} />
              </div>
            </div>
          </div>
        </div>

        {/* Override Rules */}
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-sm font-semibold mb-4">Override Rules</h2>
          <div className="space-y-2">
            {(data?.override_rules ?? []).map((r: any, i: number) => (
              <div key={i} className="flex items-center justify-between bg-gray-800 rounded-lg p-3">
                <div>
                  <p className="text-sm font-medium">{r.condition}</p>
                  <p className="text-xs text-gray-400">{r.description}</p>
                </div>
                <span className={"text-xs px-2 py-0.5 rounded " + (
                  r.action === "allow" ? "bg-green-900 text-green-300" :
                  r.action === "deny" ? "bg-red-900 text-red-300" :
                  "bg-yellow-900 text-yellow-300"
                )}>
                  {r.action}
                </span>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}
