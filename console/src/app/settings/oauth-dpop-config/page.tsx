"use client";

import { useOAuthDpopConfig } from "@ggid/sdk-react";
import { Shield, Key, AlertTriangle, Activity, Settings } from "lucide-react";

export default function OAuthDpopConfigPage() {
  const { data, loading, error, refresh, toggleRequireDpop } = useOAuthDpopConfig();

  if (loading) return <div className="p-8 text-gray-400">Loading DPoP configuration...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">OAuth DPoP Configuration</h1>
          <p className="text-sm text-gray-400 mt-1">Demonstration of Proof-of-Possession (RFC 9449) settings</p>
        </div>
        <button
          onClick={refresh}
          className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition"
        >
          Refresh
        </button>
      </div>

      {/* Global DPoP Toggle */}
      <div className="bg-gray-900 rounded-xl p-6 mb-6">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <Shield className={`w-8 h-8 ${data?.require_dpop ? "text-green-400" : "text-gray-500"}`} />
            <div>
              <h2 className="text-lg font-semibold">Require DPoP Globally</h2>
              <p className="text-sm text-gray-400">
                Enforce proof-of-possession for all token requests
              </p>
            </div>
          </div>
          <button
            onClick={() => toggleRequireDpop(!data?.require_dpop)}
            className={`relative w-14 h-7 rounded-full transition ${
              data?.require_dpop ? "bg-green-600" : "bg-gray-700"
            }`}
          >
            <span
              className={`absolute top-0.5 left-0.5 w-6 h-6 rounded-full bg-white transition-transform ${
                data?.require_dpop ? "translate-x-7" : ""
              }`}
            />
          </button>
        </div>
      </div>

      {/* Config Settings */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-6">
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-blue-400">
            <Settings className="w-4 h-4" />
            <span className="text-xs text-gray-400">Proof Max Age</span>
          </div>
          <p className="text-xl font-bold">{data?.proof_max_age_seconds ?? 60}s</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-purple-400">
            <Key className="w-4 h-4" />
            <span className="text-xs text-gray-400">Key Binding Algorithm</span>
          </div>
          <p className="text-xl font-bold">{data?.key_binding_algorithm ?? "ES256"}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-green-400">
            <Activity className="w-4 h-4" />
            <span className="text-xs text-gray-400">DPoP-Bound Tokens (24h)</span>
          </div>
          <p className="text-xl font-bold">{data?.dpop_stats?.tokens_bound_24h ?? 0}</p>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* DPoP Stats Detail */}
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-lg font-semibold mb-4">DPoP Statistics</h2>
          <div className="space-y-3">
            <StatRow label="Proofs Validated (24h)" value={data?.dpop_stats?.proofs_validated_24h ?? 0} />
            <StatRow label="Proofs Rejected (24h)" value={data?.dpop_stats?.proofs_rejected_24h ?? 0} />
            <StatRow label="Replay Attempts Blocked" value={data?.dpop_stats?.replay_blocked ?? 0} />
            <StatRow label="Avg Validation Latency" value={`${data?.dpop_stats?.avg_latency_ms ?? 0}ms`} />
            <StatRow label="Non-CE Clients Detected" value={data?.dpop_stats?.non_confidential_clients ?? 0} />
          </div>
        </div>

        {/* Per-Client Overrides + Exemptions */}
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-lg font-semibold mb-4">Per-Client Overrides</h2>
          <div className="space-y-2 mb-6">
            {(data?.per_client_overrides ?? []).map((client) => (
              <div key={client.client_id} className="bg-gray-800 rounded-lg p-3">
                <div className="flex items-center justify-between mb-1">
                  <span className="text-sm font-medium">{client.client_name}</span>
                  <span
                    className={`text-xs px-2 py-0.5 rounded ${
                      client.dpop_required
                        ? "bg-green-900 text-green-300"
                        : "bg-yellow-900 text-yellow-300"
                    }`}
                  >
                    {client.dpop_required ? "Required" : "Optional"}
                  </span>
                </div>
                <p className="text-xs text-gray-400 font-mono">{client.client_id}</p>
              </div>
            ))}
            {(data?.per_client_overrides ?? []).length === 0 && (
              <p className="text-sm text-gray-500 text-center py-4">No per-client overrides configured.</p>
            )}
          </div>

          <div className="pt-4 border-t border-gray-800">
            <h3 className="text-sm font-semibold flex items-center gap-2 mb-3">
              <AlertTriangle className="w-4 h-4 text-yellow-400" />
              Exempted Clients
            </h3>
            <div className="space-y-1">
              {(data?.exempted_clients ?? []).map((client) => (
                <div key={client.client_id} className="flex items-center justify-between bg-gray-800 rounded px-3 py-1.5">
                  <span className="text-xs text-gray-300">{client.client_name}</span>
                  <span className="text-xs text-gray-500 font-mono">{client.client_id}</span>
                </div>
              ))}
              {(data?.exempted_clients ?? []).length === 0 && (
                <p className="text-xs text-gray-500 text-center py-2">No exempted clients.</p>
              )}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

function StatRow({ label, value }: { label: string; value: string | number }) {
  return (
    <div className="flex items-center justify-between bg-gray-800 rounded-lg p-3">
      <span className="text-sm text-gray-400">{label}</span>
      <span className="text-sm font-semibold">{typeof value === "number" ? value.toLocaleString() : value}</span>
    </div>
  );
}
