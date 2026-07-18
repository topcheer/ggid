"use client";

import { useLateralMovementDetect } from "@ggid/sdk-react";
import { Network, ShieldAlert, Activity, Target } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

export default function LateralMovementDetectPage() {
  const t = useTranslations();

  const { data, loading, error, refresh } = useLateralMovementDetect();

  if (loading) return <div className="p-8 text-gray-400">Loading lateral movement detection...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">Lateral Movement Detection</h1>
          <p className="text-sm text-gray-400 mt-1">Detect adversary lateral movement across resources</p>
        </div>
        <button onClick={refresh} className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition">Refresh</button>
      </div>

      {/* Summary */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
        <div className="bg-gray-900 rounded-xl p-4">
          <Network className="w-5 h-5 text-red-400 mb-1" />
          <p className="text-xs text-gray-400">Detected Patterns</p>
          <p className="text-xl font-bold text-red-400">{data?.detected_patterns?.length ?? 0}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <Target className="w-5 h-5 text-yellow-400 mb-1" />
          <p className="text-xs text-gray-400">Kill Chain Stage</p>
          <p className="text-sm font-bold capitalize">{data?.detected_patterns?.[0]?.kill_chain_stage ?? "none"}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <Activity className="w-5 h-5 text-blue-400 mb-1" />
          <p className="text-xs text-gray-400">Avg Confidence</p>
          <p className="text-xl font-bold">
            {data?.detected_patterns?.length ? Math.round(data.detected_patterns.reduce((a, p) => a + p.confidence_score, 0) / data.detected_patterns.length * 100) : 0}%
          </p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <ShieldAlert className="w-5 h-5 text-purple-400 mb-1" />
          <p className="text-xs text-gray-400">Active Investigations</p>
          <p className="text-xl font-bold">{data?.detected_patterns?.filter((p) => p.status === "investigating").length ?? 0}</p>
        </div>
      </div>

      {/* Detected Patterns */}
      <div className="bg-gray-900 rounded-xl p-6">
        <h2 className="text-lg font-semibold mb-4">Detected Lateral Movement Patterns</h2>
        <div className="space-y-3">
          {(data?.detected_patterns ?? []).map((p) => (
            <div key={p.id} className="bg-gray-800 rounded-lg p-4">
              <div className="flex items-start justify-between mb-3">
                <div>
                  <p className="text-sm font-semibold">{p.user}</p>
                  <p className="text-xs text-gray-400">Access velocity: {p.access_velocity} resources/min</p>
                </div>
                <div className="flex items-center gap-2">
                  <div className="flex items-center gap-1">
                    <div className="w-12 h-1.5 bg-gray-700 rounded-full">
                      <div className={"h-full rounded-full " + (p.confidence_score > 0.8 ? "bg-red-500" : "bg-yellow-500")} style={{ width: (p.confidence_score * 100) + "%" }} />
                    </div>
                    <span className="text-xs">{(p.confidence_score * 100).toFixed(0)}%</span>
                  </div>
                  <span className={"text-xs px-2 py-0.5 rounded " + (
                    p.status === "investigating" ? "bg-yellow-900 text-yellow-300" :
                    p.status === "blocked" ? "bg-red-900 text-red-300" :
                    "bg-green-900 text-green-300"
                  )}>{p.status}</span>
                </div>
              </div>
              {/* Resource Chain Visual */}
              <div className="flex items-center gap-1 mb-3 overflow-x-auto pb-1">
                {p.resource_chain.map((r: any, i: number) => (
                  <div key={i} className="flex items-center gap-1 flex-shrink-0">
                    <span className="text-xs px-2 py-1 bg-gray-700 rounded font-mono text-gray-300">{r}</span>
                    {i < p.resource_chain.length - 1 && <span className="text-gray-600">{" -> "}</span>}
                  </div>
                ))}
              </div>
              <div className="flex items-center justify-between">
                <div className="flex flex-wrap gap-1">
                  {p.mitre_techniques.map((t) => (
                    <span key={t} className="text-xs px-1.5 py-0.5 bg-purple-900/50 text-purple-300 rounded font-mono">{t}</span>
                  ))}
                </div>
                <div className="flex items-center gap-2 text-xs text-gray-500">
                  <span className="px-1.5 py-0.5 bg-orange-900/50 text-orange-300 rounded">{p.kill_chain_stage}</span>
                  <span>{p.timeline}</span>
                </div>
              </div>
            </div>
          ))}
          {(data?.detected_patterns?.length ?? 0) === 0 && (
            <p className="text-sm text-gray-500">No lateral movement detected</p>
          )}
        </div>
      </div>
    </div>
  );
}
