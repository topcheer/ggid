"use client";

import { useAuthTokenBindingEnforcement } from "@ggid/sdk-react";
import { Shield, ShieldAlert, ShieldCheck, Clock, AlertTriangle, Zap } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

export default function AuthTokenBindingEnforcementPage() {
  const t = useTranslations();

  const { data, loading, error, refresh } = useAuthTokenBindingEnforcement();

  if (loading) return <div className="p-8 text-gray-400">Loading token binding enforcement...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  const levelColors: Record<string, string> = {
    none: "bg-gray-700 text-gray-400",
    optional: "bg-blue-900 text-blue-300",
    required: "bg-yellow-900 text-yellow-300",
    strict: "bg-red-900 text-red-300",
  };

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">Token Binding Enforcement</h1>
          <p className="text-sm text-gray-400 mt-1">Enforce sender-constrained tokens (DPoP, mTLS, PKI)</p>
        </div>
        <button onClick={refresh} className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition">Refresh</button>
      </div>

      {/* Summary */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
        <div className="bg-gray-900 rounded-xl p-4">
          <Shield className="w-5 h-5 text-blue-400 mb-1" />
          <p className="text-xs text-gray-400">Enforcement Level</p>
          <span className={"inline-block mt-1 text-sm font-bold px-2 py-0.5 rounded " + (levelColors[data?.enforcement_level ?? "none"] ?? "bg-gray-700")}>
            {data?.enforcement_level ?? "none"}
          </span>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <Clock className="w-5 h-5 text-yellow-400 mb-1" />
          <p className="text-xs text-gray-400">Grace Period</p>
          <p className="text-lg font-bold">{data?.grace_period_days ?? 0} days</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <ShieldAlert className="w-5 h-5 text-red-400 mb-1" />
          <p className="text-xs text-gray-400">Non-Compliant Tokens</p>
          <p className="text-lg font-bold text-red-400">{data?.non_compliant_tokens_count ?? 0}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <Zap className="w-5 h-5 text-purple-400 mb-1" />
          <p className="text-xs text-gray-400">Auto-Revoke</p>
          <p className="text-lg font-bold">{data?.auto_revoke_enabled ? "On" : "Off"}</p>
        </div>
      </div>

      {/* Per-Client Binding Policy */}
      <div className="bg-gray-900 rounded-xl p-6 mb-6">
        <h2 className="text-lg font-semibold mb-4">Per-Client Binding Policy</h2>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-gray-800 text-gray-400">
                <th scope="col" className="text-left py-2 pr-3">Client ID</th>
                <th scope="col" className="text-left py-2 pr-3">Min Binding Strength</th>
                <th scope="col" className="text-left py-2 pr-3">Allowed Methods</th>
                <th scope="col" className="text-left py-2 pr-3">Status</th>
              </tr>
            </thead>
            <tbody>
              {(data?.per_client_binding_policy ?? []).map((c) => (
                <tr key={c.client_id} className="border-b border-gray-800">
                  <td className="py-3 pr-3 font-mono text-xs text-blue-400">{c.client_id}</td>
                  <td className="py-3 pr-3">
                    <span className={"text-xs px-2 py-0.5 rounded " + levelColors[c.min_binding_strength]}>
                      {c.min_binding_strength}
                    </span>
                  </td>
                  <td className="py-3 pr-3">
                    <div className="flex flex-wrap gap-1">
                      {c.allowed_methods.map((m) => (
                        <span key={m} className="text-xs px-1.5 py-0.5 bg-gray-800 rounded font-mono">{m}</span>
                      ))}
                    </div>
                  </td>
                  <td className="py-3 pr-3">
                    <span className={"flex items-center gap-1 text-xs " + (c.non_compliant_count > 0 ? "text-red-400" : "text-green-400")}>
                      {c.non_compliant_count > 0 ? <AlertTriangle className="w-3 h-3" /> : <ShieldCheck className="w-3 h-3" />}
                      {c.non_compliant_count > 0 ? c.non_compliant_count + " violations" : "Compliant"}
                    </span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      {/* Migration Timeline */}
      <div className="bg-gray-900 rounded-xl p-6">
        <h2 className="text-lg font-semibold mb-4">Migration Timeline</h2>
        <div className="space-y-3">
          {(data?.migration_timeline ?? []).map((phase, i) => (
            <div key={i} className="flex items-center gap-4">
              <div className={"w-8 h-8 rounded-full flex items-center justify-center text-xs font-bold " + (
                phase.status === "completed" ? "bg-green-600" :
                phase.status === "active" ? "bg-blue-600" :
                "bg-gray-700"
              )}>
                {phase.status === "completed" ? <ShieldCheck className="w-4 h-4" /> : i + 1}
              </div>
              <div className="flex-1">
                <p className="text-sm font-medium">{phase.phase}</p>
                <p className="text-xs text-gray-400">{phase.description}</p>
              </div>
              <span className={"text-xs px-2 py-0.5 rounded " + (
                phase.status === "completed" ? "bg-green-900 text-green-300" :
                phase.status === "active" ? "bg-blue-900 text-blue-300" :
                "bg-gray-800 text-gray-500"
              )}>
                {phase.status}
              </span>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
