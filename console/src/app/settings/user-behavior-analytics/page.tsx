"use client";

import { useUserBehaviorAnalytics } from "@ggid/sdk-react";
import { Activity, AlertTriangle, Smartphone, MapPin, Clock, TrendingUp, TrendingDown } from "lucide-react";

export default function UserBehaviorAnalyticsPage() {
  const { data, loading, error, refresh } = useUserBehaviorAnalytics();

  if (loading) return <div className="p-8 text-gray-400">Loading user behavior analytics...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  const devScore = data?.deviation_score ?? 0;
  const scoreColor =
    devScore >= 70 ? "text-red-400" : devScore >= 40 ? "text-yellow-400" : "text-green-400";

  const anomalyIcons: Record<string, React.ReactNode> = {
    unusual_time: <Clock className="w-4 h-4 text-yellow-400" />,
    new_device: <Smartphone className="w-4 h-4 text-orange-400" />,
    new_location: <MapPin className="w-4 h-4 text-red-400" />,
    unusual_access: <AlertTriangle className="w-4 h-4 text-red-400" />,
  };

  const maxTrend = Math.max(...(data?.trend_7d ?? [1]), 1);

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">User Behavior Analytics</h1>
          <p className="text-sm text-gray-400 mt-1">Behavioral baselines and anomaly detection for user activities</p>
        </div>
        <button
          onClick={refresh}
          className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition"
        >
          Refresh
        </button>
      </div>

      {/* Deviation Score + Baseline */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6 mb-6">
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-sm text-gray-400 mb-2">Deviation Score</h2>
          <p className={`text-5xl font-bold ${scoreColor}`}>{devScore}</p>
          <div className="mt-3 w-full bg-gray-700 rounded-full h-2">
            <div
              className={devScore >= 70 ? "bg-red-500" : devScore >= 40 ? "bg-yellow-500" : "bg-green-500"}
              style={{ width: `${devScore}%`, height: "100%", borderRadius: "9999px" }}
            />
          </div>
          <p className="text-xs text-gray-500 mt-2">
            {devScore >= 70 ? "High deviation - investigate" : devScore >= 40 ? "Moderate deviation" : "Within normal range"}
          </p>
        </div>

        <div className="bg-gray-900 rounded-xl p-6 lg:col-span-2">
          <h2 className="text-sm font-semibold mb-4">Behavioral Baseline</h2>
          <div className="grid grid-cols-2 gap-4">
            <div className="bg-gray-800 rounded-lg p-3">
              <div className="flex items-center gap-2 mb-1 text-gray-400">
                <Clock className="w-3 h-3" />
                <span className="text-xs">Login Time Range</span>
              </div>
              <p className="text-sm font-medium">{data?.baseline?.login_time_range ?? "N/A"}</p>
            </div>
            <div className="bg-gray-800 rounded-lg p-3">
              <div className="flex items-center gap-2 mb-1 text-gray-400">
                <Smartphone className="w-3 h-3" />
                <span className="text-xs">Typical Devices</span>
              </div>
              <p className="text-sm font-medium">{(data?.baseline?.typical_devices ?? []).join(", ")}</p>
            </div>
            <div className="bg-gray-800 rounded-lg p-3">
              <div className="flex items-center gap-2 mb-1 text-gray-400">
                <MapPin className="w-3 h-3" />
                <span className="text-xs">Geo Patterns</span>
              </div>
              <p className="text-sm font-medium">{(data?.baseline?.geo_patterns ?? []).join(", ")}</p>
            </div>
            <div className="bg-gray-800 rounded-lg p-3">
              <div className="flex items-center gap-2 mb-1 text-gray-400">
                <Activity className="w-3 h-3" />
                <span className="text-xs">Access Patterns</span>
              </div>
              <p className="text-sm font-medium">{(data?.baseline?.access_patterns ?? []).join(", ")}</p>
            </div>
          </div>
        </div>
      </div>

      {/* Anomalies + Trend */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
            <AlertTriangle className="w-5 h-5 text-yellow-400" />
            Anomalies Detected (7d)
          </h2>
          <div className="space-y-2">
            {(data?.anomalies ?? []).map((a, i) => (
              <div key={i} className="flex items-center gap-3 bg-gray-800 rounded-lg p-3">
                {anomalyIcons[a.type] ?? <AlertTriangle className="w-4 h-4 text-gray-400" />}
                <div className="flex-1">
                  <p className="text-sm font-medium">{a.description}</p>
                  <p className="text-xs text-gray-400">{a.timestamp}</p>
                </div>
                <span
                  className={"text-xs px-2 py-0.5 rounded " + (
                    a.severity === "high" ? "bg-red-900 text-red-300" :
                    a.severity === "medium" ? "bg-yellow-900 text-yellow-300" :
                    "bg-blue-900 text-blue-300"
                  )}
                >
                  {a.severity}
                </span>
              </div>
            ))}
            {(data?.anomalies ?? []).length === 0 && (
              <p className="text-sm text-gray-500 text-center py-4">No anomalies detected.</p>
            )}
          </div>
        </div>

        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-lg font-semibold mb-4">Deviation Trend (7d)</h2>
          <div className="flex items-end gap-2 h-40">
            {(data?.trend_7d ?? []).map((score, i) => (
              <div key={i} className="flex-1 flex flex-col items-center gap-1">
                <div
                  className={
                    "w-full rounded-t " +
                    (score >= 70 ? "bg-red-500" : score >= 40 ? "bg-yellow-500" : "bg-green-500")
                  }
                  style={{ height: `${(score / maxTrend) * 100}%`, minHeight: "4px" }}
                  title={`Day ${i + 1}: ${score}`}
                />
                <span className="text-xs text-gray-500">D{i + 1}</span>
              </div>
            ))}
          </div>
          <div className="flex items-center gap-2 mt-4 pt-4 border-t border-gray-800">
            {data && data.trend_7d && data.trend_7d.length >= 2 &&
              data.trend_7d[data.trend_7d.length - 1] >= data.trend_7d[0] ? (
              <TrendingUp className="w-4 h-4 text-red-400" />
            ) : (
              <TrendingDown className="w-4 h-4 text-green-400" />
            )}
            <span className="text-sm text-gray-400">7-day trend</span>
          </div>
        </div>
      </div>
    </div>
  );
}
