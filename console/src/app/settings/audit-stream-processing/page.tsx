"use client";

import { useAuditStreamProcessing } from "@ggid/sdk-react";
import { useTranslations } from "@/lib/i18n";
import { Activity, Zap, AlertTriangle, RefreshCw, TrendingDown, Server, Clock } from "lucide-react";

export default function AuditStreamProcessingPage() {
  const t = useTranslations();
  const { data, loading, error, refresh } = useAuditStreamProcessing();

  if (loading) return <div className="p-8 text-gray-400">Loading stream processing...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  const lagHistory = data?.consumer_lag_history ?? [];
  const maxLag = Math.max(...lagHistory.map((l) => l.lag_seconds), 1);

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">Audit Stream Processing</h1>
          <p className="text-sm text-gray-400 mt-1">Real-time audit event stream health, consumer lag, and scaling</p>
        </div>
        <button
          onClick={refresh}
          className="flex items-center gap-2 px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition"
        >
          <RefreshCw className="w-4 h-4" />
          Refresh
        </button>
      </div>

      {/* Stream Health */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-green-400">
            <Zap className="w-4 h-4" />
            <span className="text-xs text-gray-400">Input Rate</span>
          </div>
          <p className="text-xl font-bold text-green-400">{data?.stream_health.input_rate.toLocaleString() ?? 0} ev/s</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-blue-400">
            <Activity className="w-4 h-4" />
            <span className="text-xs text-gray-400">Processing Rate</span>
          </div>
          <p className="text-xl font-bold text-blue-400">{data?.stream_health.processing_rate.toLocaleString() ?? 0} ev/s</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-purple-400">
            <TrendingDown className="w-4 h-4" />
            <span className="text-xs text-gray-400">Output Rate</span>
          </div>
          <p className="text-xl font-bold">{data?.stream_health.output_rate.toLocaleString() ?? 0} ev/s</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-yellow-400">
            <Clock className="w-4 h-4" />
            <span className="text-xs text-gray-400">Lag</span>
          </div>
          <p className={"text-xl font-bold " + ((data?.stream_health.lag_seconds ?? 0) > 10 ? "text-red-400" : "text-green-400")}>
            {data?.stream_health.lag_seconds ?? 0}s
          </p>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Consumer Lag Chart */}
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-lg font-semibold mb-4">Consumer Lag (60s)</h2>
          <div className="flex items-end gap-1 h-40">
            {lagHistory.map((point, i) => (
              <div
                key={i}
                className="flex-1 rounded-t"
                style={{
                  height: `${(point.lag_seconds / maxLag) * 100}%`,
                  backgroundColor: point.lag_seconds > 10 ? "#ef4444" : point.lag_seconds > 5 ? "#eab308" : "#22c55e",
                  minHeight: "2px",
                }}
                title={`${point.lag_seconds}s @ ${point.timestamp}`}
              />
            ))}
          </div>
          <div className="flex justify-between text-xs text-gray-500 mt-1">
            <span>60s ago</span>
            <span>now</span>
          </div>
        </div>

        <div className="space-y-6">
          {/* Dead Letter Queue & Backpressure */}
          <div className="grid grid-cols-2 gap-3">
            <div className="bg-gray-900 rounded-xl p-4">
              <div className="flex items-center gap-2 mb-1 text-red-400">
                <AlertTriangle className="w-4 h-4" />
                <span className="text-xs text-gray-400">Dead Letter Queue</span>
              </div>
              <p className="text-2xl font-bold text-red-400">{data?.dead_letter_queue_count ?? 0}</p>
            </div>
            <div className="bg-gray-900 rounded-xl p-4">
              <div className="flex items-center gap-2 mb-1 text-yellow-400">
                <Server className="w-4 h-4" />
                <span className="text-xs text-gray-400">Backpressure</span>
              </div>
              <p className={"text-sm font-bold capitalize " + (
                data?.backpressure_status === "critical" ? "text-red-400" :
                data?.backpressure_status === "warning" ? "text-yellow-400" :
                "text-green-400"
              )}>
                {data?.backpressure_status ?? "normal"}
              </p>
            </div>
          </div>

          {/* Retry Policy */}
          <div className="bg-gray-900 rounded-xl p-6">
            <h2 className="text-sm font-semibold mb-3">Retry Policy</h2>
            <div className="space-y-2">
              <div className="flex items-center justify-between bg-gray-800 rounded-lg p-2">
                <span className="text-xs text-gray-300">Max Retries</span>
                <span className="text-sm font-medium">{data?.retry_policy.max_retries ?? 0}</span>
              </div>
              <div className="flex items-center justify-between bg-gray-800 rounded-lg p-2">
                <span className="text-xs text-gray-300">Backoff Strategy</span>
                <span className="text-sm font-medium capitalize">{data?.retry_policy.backoff_strategy ?? "exponential"}</span>
              </div>
              <div className="flex items-center justify-between bg-gray-800 rounded-lg p-2">
                <span className="text-xs text-gray-300">Initial Delay</span>
                <span className="text-sm font-medium">{data?.retry_policy.initial_delay_ms ?? 0}ms</span>
              </div>
              <div className="flex items-center justify-between bg-gray-800 rounded-lg p-2">
                <span className="text-xs text-gray-300">Max Delay</span>
                <span className="text-sm font-medium">{data?.retry_policy.max_delay_ms ?? 0}ms</span>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Scaling Config */}
      <div className="bg-gray-900 rounded-xl p-6 mt-6">
        <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
          <Server className="w-5 h-5 text-purple-400" />
          Scaling Configuration
        </h2>
        <div className="grid grid-cols-1 md:grid-cols-4 gap-3">
          <div className="bg-gray-800 rounded-lg p-3">
            <p className="text-xs text-gray-400 mb-1">Auto-Scale Threshold</p>
            <p className="text-sm font-medium">{data?.scaling_config.auto_scale_threshold ?? 0} ev/s</p>
          </div>
          <div className="bg-gray-800 rounded-lg p-3">
            <p className="text-xs text-gray-400 mb-1">Min Consumers</p>
            <p className="text-sm font-medium">{data?.scaling_config.min_consumers ?? 0}</p>
          </div>
          <div className="bg-gray-800 rounded-lg p-3">
            <p className="text-xs text-gray-400 mb-1">Max Consumers</p>
            <p className="text-sm font-medium">{data?.scaling_config.max_consumers ?? 0}</p>
          </div>
          <div className="bg-gray-800 rounded-lg p-3">
            <p className="text-xs text-gray-400 mb-1">Current Consumers</p>
            <p className="text-sm font-bold text-blue-400">{data?.scaling_config.current_consumers ?? 0}</p>
          </div>
        </div>
      </div>
    </div>
  );
}
