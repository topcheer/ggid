"use client";

import { useAgentCredentialRotation } from "@ggid/sdk-react";
import { KeyRound, RotateCw, CheckCircle, Clock, Shield } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

export default function AgentCredentialRotationPage() {
  const t = useTranslations();

  const { data, loading, error, refresh, rotateNow } = useAgentCredentialRotation();

  if (loading) return <div className="p-8 text-gray-400">Loading credential rotation...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">Agent Credential Rotation</h1>
          <p className="text-sm text-gray-400 mt-1">Manage agent credential lifecycle and rotation schedule</p>
        </div>
        <button onClick={refresh} className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition">Refresh</button>
      </div>

      {/* Summary */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
        <div className="bg-gray-900 rounded-xl p-4">
          <KeyRound className="w-5 h-5 text-blue-400 mb-1" />
          <p className="text-xs text-gray-400">Total Agents</p>
          <p className="text-xl font-bold">{data?.rotation_schedule?.length ?? 0}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <Clock className="w-5 h-5 text-yellow-400 mb-1" />
          <p className="text-xs text-gray-400">Rotation Due</p>
          <p className="text-xl font-bold text-yellow-400">{data?.rotation_schedule?.filter((r) => r.rotation_due_days <= 0).length ?? 0}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <CheckCircle className="w-5 h-5 text-green-400 mb-1" />
          <p className="text-xs text-gray-400">Compliance</p>
          <p className="text-xl font-bold text-green-400">{data?.compliance_pct ?? 0}%</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <Shield className="w-5 h-5 text-purple-400 mb-1" />
          <p className="text-xs text-gray-400">Auto-Rotate</p>
          <p className="text-sm font-bold">{data?.rotation_schedule?.filter((r) => r.auto_rotate).length ?? 0} agents</p>
        </div>
      </div>

      {/* Compliance Gauge */}
      <div className="bg-gray-900 rounded-xl p-6 mb-6">
        <div className="flex items-center gap-6">
          <div className="relative w-20 h-20">
            <svg className="w-20 h-20 -rotate-90" viewBox="0 0 100 100">
              <circle cx="50" cy="50" r="40" fill="none" stroke="#374151" strokeWidth="10" />
              <circle cx="50" cy="50" r="40" fill="none" stroke={data?.compliance_pct === 100 ? "#22c55e" : "#eab308"} strokeWidth="10" strokeDasharray={((data?.compliance_pct ?? 0) / 100 * 251.2) + " " + 251.2} strokeLinecap="round" />
            </svg>
            <div className="absolute inset-0 flex items-center justify-center">
              <span className="text-lg font-bold text-green-400">{data?.compliance_pct ?? 0}%</span>
            </div>
          </div>
          <div>
            <h2 className="text-sm font-semibold">Rotation Compliance Score</h2>
            <p className="text-xs text-gray-400">{data?.rotation_schedule?.filter((r) => r.rotation_due_days > 0).length ?? 0} of {data?.rotation_schedule?.length ?? 0} agents within rotation window</p>
          </div>
        </div>
      </div>

      {/* Rotation Schedule Table */}
      <div className="bg-gray-900 rounded-xl p-6 mb-6">
        <h2 className="text-lg font-semibold mb-4">Rotation Schedule</h2>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-gray-800 text-gray-400">
                <th scope="col" className="text-left py-2 pr-3">Agent</th>
                <th scope="col" className="text-left py-2 pr-3">Key Age (days)</th>
                <th scope="col" className="text-left py-2 pr-3">Rotation Due (days)</th>
                <th scope="col" className="text-left py-2 pr-3">Auto-Rotate</th>
                <th scope="col" className="text-left py-2 pr-3">Actions</th>
              </tr>
            </thead>
            <tbody>
              {(data?.rotation_schedule ?? []).map((r) => (
                <tr key={r.agent_id} className="border-b border-gray-800">
                  <td className="py-3 pr-3 text-sm font-medium">{r.agent_name}</td>
                  <td className="py-3 pr-3 text-xs">{r.current_key_age_days}</td>
                  <td className="py-3 pr-3">
                    <span className={"text-xs px-2 py-0.5 rounded " + (
                      r.rotation_due_days <= 0 ? "bg-red-900 text-red-300" :
                      r.rotation_due_days <= 7 ? "bg-yellow-900 text-yellow-300" :
                      "bg-green-900 text-green-300"
                    )}>
                      {r.rotation_due_days <= 0 ? "Overdue" : r.rotation_due_days + "d"}
                    </span>
                  </td>
                  <td className="py-3 pr-3">
                    <span className={"text-xs " + (r.auto_rotate ? "text-green-400" : "text-gray-400")}>
                      {r.auto_rotate ? "Yes" : "No"}
                    </span>
                  </td>
                  <td className="py-3 pr-3">
                    <button
                      onClick={() => rotateNow(r.agent_id)}
                      className="flex items-center gap-1 text-xs text-blue-400 hover:underline"
                    >
                      <RotateCw className="w-3 h-3" /> Rotate Now
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      {/* Rotation History */}
      <div className="bg-gray-900 rounded-xl p-6">
        <h2 className="text-sm font-semibold flex items-center gap-2 mb-4">
          <RotateCw className="w-4 h-4 text-blue-400" />
          Rotation History
        </h2>
        <div className="space-y-2">
          {(data?.rotation_history ?? []).map((h: any, i: number) => (
            <div key={i} className="flex items-center gap-3 bg-gray-800 rounded-lg p-3">
              <KeyRound className="w-3 h-3 text-blue-400" />
              <div className="flex-1">
                <p className="text-sm font-medium">{h.agent_name}</p>
                <p className="text-xs text-gray-400 font-mono">{h.key_thumbprint_before.substring(0, 16)}...{" -> "}{h.key_thumbprint_after.substring(0, 16)}...</p>
              </div>
              <div className="text-right">
                <p className="text-xs text-gray-400">{h.rotated_by}</p>
                <p className="text-xs text-gray-500">{h.rotated_at}</p>
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
