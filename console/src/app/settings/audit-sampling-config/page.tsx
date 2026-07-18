"use client";

import { useAuditSamplingConfig } from "@ggid/sdk-react";
import { useTranslations } from "@/lib/i18n";
import { FlaskConical, Target, BarChart3, CheckCircle } from "lucide-react";

export default function AuditSamplingConfigPage() {
  const t = useTranslations();
  const { data, loading, error, refresh } = useAuditSamplingConfig();

  if (loading) return <div className="p-8 text-gray-400">Loading sampling config...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">Audit Sampling Configuration</h1>
          <p className="text-sm text-gray-400 mt-1">Configure audit event sampling strategies and review rates</p>
        </div>
        <button
          onClick={refresh}
          className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition"
        >
          Refresh
        </button>
      </div>

      {/* Population Stats */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-blue-400">
            <BarChart3 className="w-4 h-4" />
            <span className="text-xs text-gray-400">Total Events (24h)</span>
          </div>
          <p className="text-xl font-bold">{data?.population_stats.total_events.toLocaleString() ?? 0}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-green-400">
            <CheckCircle className="w-4 h-4" />
            <span className="text-xs text-gray-400">Sampled</span>
          </div>
          <p className="text-xl font-bold text-green-400">{data?.population_stats.sampled.toLocaleString() ?? 0}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-gray-400">
            <FlaskConical className="w-4 h-4" />
            <span className="text-xs text-gray-400">Unsampled</span>
          </div>
          <p className="text-xl font-bold">{data?.population_stats.unsampled.toLocaleString() ?? 0}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-purple-400">
            <Target className="w-4 h-4" />
            <span className="text-xs text-gray-400">Confidence Interval</span>
          </div>
          <p className="text-xl font-bold">{data?.confidence_interval_target ?? 0}%</p>
        </div>
      </div>

      {/* Sampling Strategies Table */}
      <div className="bg-gray-900 rounded-xl p-6 mb-6">
        <h2 className="text-lg font-semibold mb-4">Sampling Strategies</h2>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-gray-800 text-gray-400">
                <th scope="col" className="text-left py-2 pr-3">Strategy</th>
                <th scope="col" className="text-left py-2 pr-3">Sample Size %</th>
                <th scope="col" className="text-left py-2 pr-3">Target Population</th>
                <th scope="col" className="text-left py-2 pr-3">Last Review</th>
              </tr>
            </thead>
            <tbody>
              {(data?.sampling_strategies ?? []).map((s: any) => (
                <tr key={s.strategy} className="border-b border-gray-800">
                  <td className="py-3 pr-3">
                    <span className={"text-xs px-2 py-1 rounded capitalize " + (
                      s.strategy === "random" ? "bg-blue-900 text-blue-300" :
                      s.strategy === "stratified" ? "bg-purple-900 text-purple-300" :
                      s.strategy === "risk_weighted" ? "bg-red-900 text-red-300" :
                      "bg-yellow-900 text-yellow-300"
                    )}>
                      {s.strategy.replace(/_/g, " ")}
                    </span>
                  </td>
                  <td className="py-3 pr-3">
                    <div className="flex items-center gap-2">
                      <div className="w-20 bg-gray-700 rounded-full h-1.5">
                        <div className="bg-blue-500 rounded-full h-1.5" style={{ width: `${s.sample_size_pct}%` }} />
                      </div>
                      <span className="text-sm font-medium">{s.sample_size_pct}%</span>
                    </div>
                  </td>
                  <td className="py-3 pr-3 text-gray-300 text-xs">{s.target_population}</td>
                  <td className="py-3 pr-3 text-gray-400 text-xs">{s.last_review ?? "Never"}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      {/* Per-Event Type Sampling Rate */}
      <div className="bg-gray-900 rounded-xl p-6">
        <h2 className="text-lg font-semibold mb-4">Per-Event Type Sampling Rate</h2>
        <div className="space-y-2">
          {(data?.per_event_type_rate ?? []).map((e: any) => (
            <div key={e.event_type} className="flex items-center gap-3 bg-gray-800 rounded-lg p-2">
              <span className="text-sm font-mono text-blue-400 w-32">{e.event_type}</span>
              <div className="flex-1 bg-gray-700 rounded-full h-2">
                <div
                  className={e.sampling_rate === 1 ? "bg-green-500" : e.sampling_rate >= 0.1 ? "bg-blue-500" : "bg-yellow-500"}
                  style={{ width: `${e.sampling_rate * 100}%`, height: "100%", borderRadius: "9999px" }}
                />
              </div>
              <span className="text-xs font-medium w-12 text-right">
                {e.sampling_rate === 1 ? "100%" : `${(e.sampling_rate * 100).toFixed(1)}%`}
              </span>
              <span className="text-xs text-gray-500 w-20 text-right">{e.volume.toLocaleString()} ev/d</span>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
