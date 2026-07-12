"use client";

import { useAuthBackChannelAuth } from "@ggid/sdk-react";
import { Smartphone, MessageSquare, Clock, Activity, Zap } from "lucide-react";

export default function AuthBackChannelAuthPage() {
  const { data, loading, error, refresh } = useAuthBackChannelAuth();

  if (loading) return <div className="p-8 text-gray-400">Loading CIBA config...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">Back-Channel Authentication (CIBA)</h1>
          <p className="text-sm text-gray-400 mt-1">Client-Initiated Backchannel Authentication Flow (RFC 9396)</p>
        </div>
        <button
          onClick={refresh}
          className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition"
        >
          Refresh
        </button>
      </div>

      {/* Config Summary */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-green-400">
            <Zap className="w-4 h-4" />
            <span className="text-xs text-gray-400">CIBA Enabled</span>
          </div>
          <p className="text-lg font-bold">{data?.enabled ? "Yes" : "No"}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-blue-400">
            <Activity className="w-4 h-4" />
            <span className="text-xs text-gray-400">Token Delivery</span>
          </div>
          <p className="text-sm font-bold capitalize">{data?.token_delivery_mode ?? "poll"}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-purple-400">
            <Clock className="w-4 h-4" />
            <span className="text-xs text-gray-400">Max Polling Interval</span>
          </div>
          <p className="text-lg font-bold">{data?.max_polling_interval_seconds ?? 0}s</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-yellow-400">
            <Clock className="w-4 h-4" />
            <span className="text-xs text-gray-400">Max Requested Expiry</span>
          </div>
          <p className="text-lg font-bold">{data?.requested_expiry_max_seconds ?? 0}s</p>
        </div>
      </div>

      {/* Binding Message */}
      <div className="bg-gray-900 rounded-xl p-6 mb-6">
        <h2 className="text-sm font-semibold flex items-center gap-2 mb-3">
          <MessageSquare className="w-4 h-4 text-blue-400" />
          Binding Message Configuration
        </h2>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <div className="bg-gray-800 rounded-lg p-3">
            <p className="text-xs text-gray-400 mb-1">Required</p>
            <p className="text-sm font-medium">{data?.binding_message_config.required ? "Yes" : "No"}</p>
          </div>
          <div className="bg-gray-800 rounded-lg p-3">
            <p className="text-xs text-gray-400 mb-1">Max Length</p>
            <p className="text-sm font-medium">{data?.binding_message_config.max_length ?? 0} chars</p>
          </div>
          <div className="bg-gray-800 rounded-lg p-3">
            <p className="text-xs text-gray-400 mb-1">Format Pattern</p>
            <p className="text-sm font-mono">{data?.binding_message_config.format_pattern ?? "^[a-zA-Z0-9 ]+$"}</p>
          </div>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Per-Client CIBA Config */}
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-lg font-semibold mb-4">Per-Client CIBA Config</h2>
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-gray-800 text-gray-400">
                  <th className="text-left py-2 pr-3">Client</th>
                  <th className="text-left py-2 pr-3">Delivery Mode</th>
                  <th className="text-left py-2 pr-3">Enabled</th>
                </tr>
              </thead>
              <tbody>
                {(data?.per_client_ciba ?? []).map((c) => (
                  <tr key={c.client_id} className="border-b border-gray-800">
                    <td className="py-3 pr-3 font-mono text-xs text-blue-400">{c.client_id}</td>
                    <td className="py-3 pr-3 capitalize">{c.delivery_mode}</td>
                    <td className="py-3 pr-3">
                      <span className={"text-xs px-2 py-0.5 rounded " + (c.enabled ? "bg-green-900 text-green-300" : "bg-gray-700 text-gray-400")}>
                        {c.enabled ? "Yes" : "No"}
                      </span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>

        {/* Usage Stats */}
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
            <Activity className="w-5 h-5 text-purple-400" />
            Usage Stats (24h)
          </h2>
          <div className="space-y-2">
            <div className="flex items-center justify-between bg-gray-800 rounded-lg p-3">
              <span className="text-sm text-gray-300">CIBA Requests</span>
              <span className="text-sm font-bold">{data?.usage_stats.ciba_requests_24h ?? 0}</span>
            </div>
            <div className="flex items-center justify-between bg-gray-800 rounded-lg p-3">
              <span className="text-sm text-gray-300">Successful Authentications</span>
              <span className="text-sm font-bold text-green-400">{data?.usage_stats.successful_24h ?? 0}</span>
            </div>
            <div className="flex items-center justify-between bg-gray-800 rounded-lg p-3">
              <span className="text-sm text-gray-300">User Rejections</span>
              <span className="text-sm font-bold text-red-400">{data?.usage_stats.rejected_24h ?? 0}</span>
            </div>
            <div className="flex items-center justify-between bg-gray-800 rounded-lg p-3">
              <span className="text-sm text-gray-300">Timeouts</span>
              <span className="text-sm font-bold text-yellow-400">{data?.usage_stats.timeouts_24h ?? 0}</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
