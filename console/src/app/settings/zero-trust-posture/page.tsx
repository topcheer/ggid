"use client";

import { useZeroTrustPosture } from "@ggid/sdk-react";
import { useTranslations } from "@/lib/i18n";import {
  ShieldCheck,
  ShieldAlert,
  Cpu,
  Lock,
  Network,
  Activity,
  TrendingUp,
  TrendingDown,
} from "lucide-react";

export default function ZeroTrustPosturePage() {
  const t = useTranslations();
  const { data, loading, error, refresh } = useZeroTrustPosture();

  if (loading) return <div className="p-8 text-gray-400">Loading zero-trust posture...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  const scoreColor =
    (data?.trust_score ?? 0) >= 80
      ? "text-green-400"
      : (data?.trust_score ?? 0) >= 50
      ? "text-yellow-400"
      : "text-red-400";

  const gaugeDeg = ((data?.trust_score ?? 0) / 100) * 180;

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">Zero-Trust Posture</h1>
          <p className="text-sm text-gray-400 mt-1">Continuous trust evaluation across identity, device, and network</p>
        </div>
        <button
          onClick={refresh}
          className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition"
        >
          Refresh
        </button>
      </div>

      {/* Trust Score Gauge */}
      <div className="bg-gray-900 rounded-xl p-6 mb-6 flex items-center gap-8">
        <div className="relative w-48 h-28">
          <svg viewBox="0 0 100 56" className="w-full h-full">
            <path d="M 10 50 A 40 40 0 0 1 90 50" fill="none" stroke="#374151" strokeWidth="8" strokeLinecap="round" />
            <path
              d="M 10 50 A 40 40 0 0 1 90 50"
              fill="none"
              stroke="currentColor"
              strokeWidth="8"
              strokeLinecap="round"
              strokeDasharray={`${gaugeDeg * 1.26} 200`}
              className={scoreColor}
            />
          </svg>
          <div className={`absolute inset-0 flex flex-col items-center justify-center ${scoreColor}`}>
            <span className="text-4xl font-bold">{data?.trust_score ?? 0}</span>
            <span className="text-xs text-gray-400">Trust Score</span>
          </div>
        </div>
        <div className="flex-1">
          <div className="flex items-center gap-2 mb-1">
            {(data?.trust_score ?? 0) >= 80 ? (
              <ShieldCheck className="w-5 h-5 text-green-400" />
            ) : (
              <ShieldAlert className="w-5 h-5 text-yellow-400" />
            )}
            <span className="text-lg font-semibold">
              {(data?.trust_score ?? 0) >= 80
                ? "Healthy"
                : (data?.trust_score ?? 0) >= 50
                ? "At Risk"
                : "Critical"}
            </span>
          </div>
          <p className="text-sm text-gray-400">
            Overall posture based on device compliance, identity verification, and network segmentation.
          </p>
        </div>
      </div>

      {/* Key Metrics Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 mb-6">
        <MetricCard
          icon={<Cpu className="w-5 h-5" />}
          label="Device Compliance"
          value={`${data?.device_compliance_pct ?? 0}%`}
          color="text-blue-400"
        />
        <MetricCard
          icon={<Lock className="w-5 h-5" />}
          label="Identity Verification"
          value={`${data?.identity_verification_rate ?? 0}%`}
          color="text-green-400"
        />
        <MetricCard
          icon={<Network className="w-5 h-5" />}
          label="Network Segmentation"
          value={`${data?.network_segmentation ?? 0}%`}
          color="text-purple-400"
        />
        <MetricCard
          icon={<Activity className="w-5 h-5" />}
          label="Continuous Auth Coverage"
          value={`${data?.continuous_auth_coverage ?? 0}%`}
          color="text-cyan-400"
        />
      </div>

      {/* Violations + Trend */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div className="bg-gray-900 rounded-xl p-6">
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-lg font-semibold">Violations (24h)</h2>
            <span className="text-2xl font-bold text-red-400">{data?.violations_24h ?? 0}</span>
          </div>
          {data?.recent_violations && data.recent_violations.length > 0 ? (
            <div className="space-y-2">
              {data.recent_violations.slice(0, 5).map((v: any, i: number) => (
                <div key={i} className="flex items-center gap-3 bg-gray-800 rounded-lg p-3">
                  <ShieldAlert className="w-4 h-4 text-red-400 flex-shrink-0" />
                  <div className="flex-1 min-w-0">
                    <p className="text-sm font-medium truncate">{v.type}</p>
                    <p className="text-xs text-gray-400">
                      {v.severity} - {v.timestamp}
                    </p>
                  </div>
                  <span className="text-xs px-2 py-0.5 rounded bg-red-900 text-red-300">
                    {v.severity}
                  </span>
                </div>
              ))}
            </div>
          ) : (
            <p className="text-sm text-gray-500 text-center py-8">No violations in the last 24 hours.</p>
          )}
        </div>

        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-lg font-semibold mb-4">Posture Trend (30d)</h2>
          <div className="flex items-end gap-1 h-40">
            {(data?.posture_trend_30d ?? []).map((t: any, i: number) => (
              <div key={i} className="flex-1 flex flex-col items-center gap-1">
                <div
                  className="w-full rounded-t bg-blue-500 hover:bg-blue-400 transition-all"
                  style={{ height: `${t}%`, minHeight: "4px" }}
                  title={`Day ${i + 1}: ${t}%`}
                />
                {i % 5 === 0 && <span className="text-xs text-gray-500">{i + 1}</span>}
              </div>
            ))}
          </div>
          <div className="flex items-center justify-between mt-4 pt-4 border-t border-gray-800">
            <div className="flex items-center gap-2">
              {data && data.posture_trend_30d && data.posture_trend_30d.length >= 2 &&
                data.posture_trend_30d[data.posture_trend_30d.length - 1] >=
                  data.posture_trend_30d[data.posture_trend_30d.length - 2] ? (
                <TrendingUp className="w-4 h-4 text-green-400" />
              ) : (
                <TrendingDown className="w-4 h-4 text-red-400" />
              )}
              <span className="text-sm text-gray-400">30-day trend</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

function MetricCard({
  icon,
  label,
  value,
  color,
}: {
  icon: React.ReactNode;
  label: string;
  value: string;
  color: string;
}) {
  return (
    <div className="bg-gray-900 rounded-xl p-4">
      <div className={`flex items-center gap-2 mb-2 ${color}`}>
        {icon}
        <span className="text-xs text-gray-400">{label}</span>
      </div>
      <p className="text-2xl font-bold">{value}</p>
    </div>
  );
}
