"use client";

import { useITDRDashboard } from "@ggid/sdk-react";
import { Shield, ShieldAlert, Activity, Zap, BookOpen, Plus } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

export default function ITDRDashboardPage() {
  const t = useTranslations();

  const { data, loading, error, refresh } = useITDRDashboard();

  if (loading) return <div className="p-8 text-gray-400">Loading ITDR dashboard...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  const sevColors: Record<string, string> = {
    critical: "bg-red-900 text-red-300",
    high: "bg-orange-900 text-orange-300",
    medium: "bg-yellow-900 text-yellow-300",
    low: "bg-blue-900 text-blue-300",
  };

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">ITDR Dashboard</h1>
          <p className="text-sm text-gray-400 mt-1">Identity Threat Detection & Response</p>
        </div>
        <button onClick={refresh} className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition">Refresh</button>
      </div>

      {/* Summary */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
        <div className="bg-gray-900 rounded-xl p-4">
          <ShieldAlert className="w-5 h-5 text-red-400 mb-1" />
          <p className="text-xs text-gray-400">Active Threats</p>
          <p className="text-xl font-bold text-red-400">{data?.threat_detections?.filter((t) => t.status === "active").length ?? 0}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <Shield className="w-5 h-5 text-green-400 mb-1" />
          <p className="text-xs text-gray-400">Detection Rules</p>
          <p className="text-xl font-bold">{data?.detection_rules?.length ?? 0}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <Activity className="w-5 h-5 text-blue-400 mb-1" />
          <p className="text-xs text-gray-400">Total Detections (7d)</p>
          <p className="text-xl font-bold">{data?.threat_detections?.length ?? 0}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <Zap className="w-5 h-5 text-purple-400 mb-1" />
          <p className="text-xs text-gray-400">Auto-Response</p>
          <p className="text-lg font-bold">{data?.auto_response_enabled ? "Enabled" : "Disabled"}</p>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Threat Detections Feed */}
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
            <ShieldAlert className="w-5 h-5 text-red-400" />
            Threat Detections
          </h2>
          <div className="space-y-2 max-h-96 overflow-y-auto">
            {(data?.threat_detections ?? []).map((t) => (
              <div key={t.id} className="bg-gray-800 rounded-lg p-3">
                <div className="flex items-start justify-between mb-2">
                  <div>
                    <p className="text-sm font-semibold">{t.type}</p>
                    <p className="text-xs text-gray-400">Source: {t.source}</p>
                  </div>
                  <span className={"text-xs px-2 py-0.5 rounded " + sevColors[t.severity]}>
                    {t.severity}
                  </span>
                </div>
                <div className="flex flex-wrap gap-1 mb-2">
                  {t.mitre_techniques.map((tech) => (
                    <span key={tech} className="text-xs px-1.5 py-0.5 bg-purple-900/50 text-purple-300 rounded font-mono">{tech}</span>
                  ))}
                </div>
                <div className="flex items-center justify-between text-xs text-gray-500">
                  <span>{t.affected_users} users affected</span>
                  <span>{t.timestamp}</span>
                  <span className={"px-1.5 py-0.5 rounded " + (
                    t.status === "active" ? "bg-red-900/50 text-red-300" :
                    t.status === "mitigated" ? "bg-yellow-900/50 text-yellow-300" :
                    "bg-green-900/50 text-green-300"
                  )}>{t.status}</span>
                </div>
              </div>
            ))}
          </div>
        </div>

        <div className="space-y-6">
          {/* Detection Rules */}
          <div className="bg-gray-900 rounded-xl p-6">
            <h2 className="text-sm font-semibold mb-4">Detection Rules</h2>
            <div className="space-y-2">
              {(data?.detection_rules ?? []).map((r) => (
                <div key={r.rule_name} className="flex items-center justify-between bg-gray-800 rounded-lg p-3">
                  <div>
                    <p className="text-sm font-medium">{r.rule_name}</p>
                    <p className="text-xs text-gray-400 font-mono">{r.technique}</p>
                  </div>
                  <div className="text-right">
                    <span className={"text-xs px-2 py-0.5 rounded " + (r.enabled ? "bg-green-900 text-green-300" : "bg-gray-700 text-gray-400")}>
                      {r.enabled ? "Active" : "Disabled"}
                    </span>
                    <p className="text-xs text-gray-500 mt-0.5">Last: {r.last_triggered}</p>
                  </div>
                </div>
              ))}
            </div>
          </div>

          {/* Response Playbooks */}
          <div className="bg-gray-900 rounded-xl p-6">
            <h2 className="text-sm font-semibold flex items-center gap-2 mb-4">
              <BookOpen className="w-4 h-4 text-blue-400" />
              Response Playbooks
            </h2>
            <div className="space-y-2">
              {(data?.response_playbooks ?? []).map((p) => (
                <div key={p.threat_type} className="flex items-center gap-3 bg-gray-800 rounded-lg p-3">
                  <div className="flex-1">
                    <p className="text-sm font-medium">{p.threat_type}</p>
                    <p className="text-xs text-gray-400">{p.steps_count} steps - {p.estimated_time}</p>
                  </div>
                  <span className="text-xs text-blue-400">{p.auto_execute ? "Auto-execute" : "Manual"}</span>
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
