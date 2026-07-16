"use client";

import { useAgentConsentFlow } from "@ggid/sdk-react";
import { Bot, CheckCircle, XCircle, Clock, Shield } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

export default function AgentConsentFlowPage() {
  const t = useTranslations();

  const { data, loading, error, refresh, approve, deny } = useAgentConsentFlow();

  if (loading) return <div className="p-8 text-gray-400">Loading agent consent...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">Agent Consent Flow</h1>
          <p className="text-sm text-gray-400 mt-1">Manage AI agent access consent requests</p>
        </div>
        <button onClick={refresh} className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition">Refresh</button>
      </div>

      {/* Summary */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
        <div className="bg-gray-900 rounded-xl p-4">
          <Clock className="w-5 h-5 text-yellow-400 mb-1" />
          <p className="text-xs text-gray-400">Pending Requests</p>
          <p className="text-xl font-bold text-yellow-400">{data?.pending_consent_requests?.length ?? 0}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <Bot className="w-5 h-5 text-blue-400 mb-1" />
          <p className="text-xs text-gray-400">Active Agents</p>
          <p className="text-xl font-bold">{data?.active_agent_count ?? 0}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <Shield className="w-5 h-5 text-green-400 mb-1" />
          <p className="text-xs text-gray-400">Granular Scope</p>
          <p className="text-sm font-bold">{data?.granular_scope_toggle ? "Enabled" : "Disabled"}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <Clock className="w-5 h-5 text-purple-400 mb-1" />
          <p className="text-xs text-gray-400">Auto-Expire</p>
          <p className="text-sm font-bold">{data?.auto_expire_hours ?? 0}h</p>
        </div>
      </div>

      {/* Pending Consent Requests */}
      <div className="bg-gray-900 rounded-xl p-6 mb-6">
        <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
          <Clock className="w-5 h-5 text-yellow-400" />
          Pending Consent Requests
        </h2>
        <div className="space-y-3">
          {(data?.pending_consent_requests ?? []).map((req) => (
            <div key={req.id} className="bg-gray-800 rounded-lg p-4">
              <div className="flex items-start justify-between mb-3">
                <div>
                  <p className="text-sm font-semibold flex items-center gap-2">
                    <Bot className="w-4 h-4 text-blue-400" />
                    {req.agent_name}
                  </p>
                  <p className="text-xs text-gray-400 mt-0.5">Resource: {req.resource} - User: {req.user}</p>
                </div>
                <span className="text-xs text-yellow-400">Expires in {req.expires_in}</span>
              </div>
              <div className="mb-3">
                <p className="text-xs text-gray-500 mb-1">Requested Scopes:</p>
                <div className="flex flex-wrap gap-1">
                  {req.requested_scopes.map((s) => (
                    <span key={s} className="text-xs px-2 py-0.5 bg-gray-700 rounded font-mono">{s}</span>
                  ))}
                </div>
              </div>
              {req.scope_justification && (
                <p className="text-xs text-gray-400 mb-3">Justification: {req.scope_justification}</p>
              )}
              <div className="flex gap-2">
                <button
                  onClick={() => approve(req.id)}
                  className="flex items-center gap-1 px-3 py-1.5 bg-green-600 hover:bg-green-700 rounded-lg text-xs font-medium transition"
                >
                  <CheckCircle className="w-3 h-3" /> Approve
                </button>
                <button
                  onClick={() => deny(req.id)}
                  className="flex items-center gap-1 px-3 py-1.5 bg-red-600 hover:bg-red-700 rounded-lg text-xs font-medium transition"
                >
                  <XCircle className="w-3 h-3" /> Deny
                </button>
              </div>
            </div>
          ))}
          {(data?.pending_consent_requests?.length ?? 0) === 0 && (
            <p className="text-sm text-gray-500">No pending consent requests</p>
          )}
        </div>
      </div>

      {/* Consent History */}
      <div className="bg-gray-900 rounded-xl p-6">
        <h2 className="text-sm font-semibold mb-4">Consent History</h2>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-gray-800 text-gray-400">
                <th scope="col" className="text-left py-2 pr-3">Agent</th>
                <th scope="col" className="text-left py-2 pr-3">User</th>
                <th scope="col" className="text-left py-2 pr-3">Scopes</th>
                <th scope="col" className="text-left py-2 pr-3">Granted</th>
                <th scope="col" className="text-left py-2 pr-3">Status</th>
              </tr>
            </thead>
            <tbody>
              {(data?.consent_history ?? []).map((h, i) => (
                <tr key={i} className="border-b border-gray-800">
                  <td className="py-3 pr-3 text-sm">{h.agent}</td>
                  <td className="py-3 pr-3 text-xs text-gray-400">{h.user}</td>
                  <td className="py-3 pr-3">
                    <div className="flex flex-wrap gap-0.5">
                      {h.scopes.map((s) => (
                        <span key={s} className="text-xs px-1 py-0.5 bg-gray-800 rounded font-mono">{s}</span>
                      ))}
                    </div>
                  </td>
                  <td className="py-3 pr-3 text-xs text-gray-400">{h.granted_at}</td>
                  <td className="py-3 pr-3">
                    <span className={"text-xs px-2 py-0.5 rounded " + (
                      h.status === "active" ? "bg-green-900 text-green-300" :
                      h.status === "revoked" ? "bg-red-900 text-red-300" :
                      "bg-gray-700 text-gray-400"
                    )}>
                      {h.status}
                    </span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
