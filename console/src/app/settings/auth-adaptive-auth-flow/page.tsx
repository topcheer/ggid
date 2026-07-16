"use client";

import { useAuthAdaptiveAuthFlow } from "@ggid/sdk-react";
import { Shield, AlertTriangle, Activity, ArrowRight, Lock } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

export default function AuthAdaptiveAuthFlowPage() {
  const t = useTranslations();

  const { data, loading, error, refresh } = useAuthAdaptiveAuthFlow();

  if (loading) return <div className="p-8 text-gray-400">Loading adaptive auth...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  const riskColors: Record<string, string> = {
    low: "bg-green-900 text-green-300",
    medium: "bg-yellow-900 text-yellow-300",
    high: "bg-orange-900 text-orange-300",
    critical: "bg-red-900 text-red-300",
  };

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">Adaptive Authentication Flow</h1>
          <p className="text-sm text-gray-400 mt-1">Risk-based step-up authentication with configurable triggers</p>
        </div>
        <button
          onClick={refresh}
          className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition"
        >
          Refresh
        </button>
      </div>

      {/* Risk Threshold Matrix */}
      <div className="bg-gray-900 rounded-xl p-6 mb-6">
        <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
          <Shield className="w-5 h-5 text-blue-400" />
          Risk Threshold Matrix
        </h2>
        <div className="grid grid-cols-1 md:grid-cols-4 gap-3">
          {(data?.risk_threshold_matrix ?? []).map((entry) => (
            <div key={entry.risk_level} className="bg-gray-800 rounded-lg p-4">
              <span className={"text-xs px-2 py-1 rounded block mb-2 text-center " + riskColors[entry.risk_level]}>
                {entry.risk_level}
              </span>
              <div className="flex items-center gap-2">
                {entry.required_factors.map((factor, i) => (
                  <span key={i} className="flex items-center gap-1">
                    <span className="text-xs px-2 py-1 bg-gray-700 rounded">{factor}</span>
                    {i < entry.required_factors.length - 1 && <span className="text-gray-500 text-xs">+</span>}
                  </span>
                ))}
              </div>
            </div>
          ))}
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Signal Weights */}
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
            <Activity className="w-5 h-5 text-purple-400" />
            Signal Weights
          </h2>
          <div className="space-y-3">
            {(data?.signal_weights ?? []).map((sig) => (
              <div key={sig.signal}>
                <div className="flex items-center justify-between mb-1">
                  <span className="text-sm capitalize">{sig.signal}</span>
                  <span className="text-xs font-medium">{(sig.weight * 100).toFixed(0)}%</span>
                </div>
                <div className="bg-gray-700 rounded-full h-2">
                  <div
                    className={sig.weight > 0.2 ? "bg-red-500" : sig.weight > 0.1 ? "bg-yellow-500" : "bg-green-500"}
                    style={{ width: `${sig.weight * 100}%`, height: "100%", borderRadius: "9999px" }}
                  />
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* Step-Up Triggers */}
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
            <AlertTriangle className="w-5 h-5 text-yellow-400" />
            Step-Up Triggers
          </h2>
          <div className="space-y-2">
            {(data?.step_up_triggers ?? []).map((t, i) => (
              <div key={i} className="flex items-center justify-between bg-gray-800 rounded-lg p-3">
                <div className="flex items-center gap-2">
                  <Lock className="w-3 h-3 text-yellow-400" />
                  <span className="text-sm capitalize">{t.action.replace(/_/g, " ")}</span>
                </div>
                <span className="text-xs px-2 py-0.5 rounded bg-gray-700 text-gray-300">{t.required_level}</span>
              </div>
            ))}
          </div>
        </div>
      </div>

      {/* Flow Visualizer */}
      <div className="bg-gray-900 rounded-xl p-6 mt-6">
        <h2 className="text-lg font-semibold mb-4">Authentication Flow</h2>
        <div className="flex items-center gap-2 flex-wrap">
          <span className="text-xs px-3 py-2 bg-blue-900 text-blue-300 rounded-lg">Request</span>
          <ArrowRight className="w-3 h-3 text-gray-500" />
          <span className="text-xs px-3 py-2 bg-gray-800 rounded-lg border border-gray-700">Risk Assessment</span>
          <ArrowRight className="w-3 h-3 text-gray-500" />
          <span className="text-xs px-3 py-2 bg-green-900 text-green-300 rounded-lg">Low: Password Only</span>
          <ArrowRight className="w-3 h-3 text-gray-500" />
          <span className="text-xs px-3 py-2 bg-yellow-900 text-yellow-300 rounded-lg">Med: +OTP</span>
          <ArrowRight className="w-3 h-3 text-gray-500" />
          <span className="text-xs px-3 py-2 bg-orange-900 text-orange-300 rounded-lg">High: +WebAuthn</span>
          <ArrowRight className="w-3 h-3 text-gray-500" />
          <span className="text-xs px-3 py-2 bg-red-900 text-red-300 rounded-lg">Critical: Deny</span>
        </div>
      </div>

      {/* Per-Role Override */}
      <div className="bg-gray-900 rounded-xl p-6 mt-6">
        <h2 className="text-lg font-semibold mb-4">Per-Role Override</h2>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-gray-800 text-gray-400">
                <th scope="col" className="text-left py-2 pr-4">Role</th>
                <th scope="col" className="text-left py-2 pr-4">Min Auth Level</th>
                <th scope="col" className="text-left py-2 pr-4">Max Session (min)</th>
              </tr>
            </thead>
            <tbody>
              {(data?.override_per_role ?? []).map((r) => (
                <tr key={r.role} className="border-b border-gray-800">
                  <td className="py-3 pr-4 font-medium">{r.role}</td>
                  <td className="py-3 pr-4">
                    <span className={"text-xs px-2 py-0.5 rounded " + riskColors[r.min_auth_level]}>{r.min_auth_level}</span>
                  </td>
                  <td className="py-3 pr-4 text-gray-300">{r.max_session_minutes}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
