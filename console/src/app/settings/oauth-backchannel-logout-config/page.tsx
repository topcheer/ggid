"use client";

import { useOAuthBackchannelLogoutConfig } from "@ggid/sdk-react";
import { LogOut, RefreshCw, AlertTriangle, CheckCircle, Clock, RotateCcw } from "lucide-react";

export default function OAuthBackchannelLogoutConfigPage() {
  const { data, loading, error, refresh, testLogout } = useOAuthBackchannelLogoutConfig();

  if (loading) return <div className="p-8 text-gray-400">Loading backchannel logout config...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">Backchannel Logout Configuration</h1>
          <p className="text-sm text-gray-400 mt-1">OIDC Back-Channel Logout (RFC 9初) endpoint and session management</p>
        </div>
        <button
          onClick={refresh}
          className="flex items-center gap-2 px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition"
        >
          <RefreshCw className="w-4 h-4" />
          Refresh
        </button>
      </div>

      {/* Endpoint Config */}
      <div className="bg-gray-900 rounded-xl p-6 mb-6">
        <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
          <LogOut className="w-5 h-5 text-blue-400" />
          Endpoint Configuration
        </h2>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div className="bg-gray-800 rounded-lg p-4">
            <label className="text-xs text-gray-400 mb-1 block">Logout Endpoint URL</label>
            <p className="text-sm font-mono text-blue-400">{data?.logout_endpoint ?? "N/A"}</p>
          </div>
          <div className="bg-gray-800 rounded-lg p-4">
            <label className="text-xs text-gray-400 mb-1 block">Session Lifetime (seconds)</label>
            <p className="text-sm font-medium">{data?.session_lifetime ?? 0}</p>
          </div>
          <div className="bg-gray-800 rounded-lg p-4">
            <label className="text-xs text-gray-400 mb-1 block">Token Revocation on Logout</label>
            <div className="flex items-center gap-2">
              {data?.token_revocation_on_logout ? (
                <CheckCircle className="w-4 h-4 text-green-400" />
              ) : (
                <AlertTriangle className="w-4 h-4 text-yellow-400" />
              )}
              <span className="text-sm">{data?.token_revocation_on_logout ? "Enabled" : "Disabled"}</span>
            </div>
          </div>
          <div className="bg-gray-800 rounded-lg p-4">
            <label className="text-xs text-gray-400 mb-1 block">Per-Client Toggle</label>
            <div className="flex items-center gap-2">
              {data?.per_client_toggle ? (
                <CheckCircle className="w-4 h-4 text-green-400" />
              ) : (
                <AlertTriangle className="w-4 h-4 text-yellow-400" />
              )}
              <span className="text-sm">{data?.per_client_toggle ? "Enabled" : "Disabled"}</span>
            </div>
          </div>
        </div>
      </div>

      {/* Error Handling */}
      <div className="bg-gray-900 rounded-xl p-6 mb-6">
        <h2 className="text-lg font-semibold mb-4">Error Handling</h2>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <div className="bg-gray-800 rounded-lg p-4">
            <div className="flex items-center gap-2 mb-1 text-blue-400">
              <RotateCcw className="w-4 h-4" />
              <span className="text-xs text-gray-400">Retry Attempts</span>
            </div>
            <p className="text-xl font-bold">{data?.error_handling?.retry_attempts ?? 0}</p>
          </div>
          <div className="bg-gray-800 rounded-lg p-4">
            <div className="flex items-center gap-2 mb-1 text-yellow-400">
              <Clock className="w-4 h-4" />
              <span className="text-xs text-gray-400">Retry Timeout</span>
            </div>
            <p className="text-xl font-bold">{data?.error_handling?.retry_timeout_seconds ?? 0}s</p>
          </div>
          <div className="bg-gray-800 rounded-lg p-4">
            <div className="flex items-center gap-2 mb-1 text-red-400">
              <AlertTriangle className="w-4 h-4" />
              <span className="text-xs text-gray-400">Failed Notifications (24h)</span>
            </div>
            <p className="text-xl font-bold">{data?.error_handling?.failed_notifications_24h ?? 0}</p>
          </div>
        </div>
      </div>

      {/* Per-Client Toggles + Test */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-lg font-semibold mb-4">Per-Client Configuration</h2>
          <div className="space-y-2">
            {(data?.client_configs ?? []).map((c) => (
              <div key={c.client_id} className="flex items-center justify-between bg-gray-800 rounded-lg p-3">
                <div>
                  <p className="text-sm font-medium">{c.client_name}</p>
                  <p className="text-xs text-gray-400 font-mono">{c.client_id}</p>
                </div>
                <span
                  className={"text-xs px-2 py-0.5 rounded " + (
                    c.backchannel_logout_enabled
                      ? "bg-green-900 text-green-300"
                      : "bg-gray-700 text-gray-400"
                  )}
                >
                  {c.backchannel_logout_enabled ? "Enabled" : "Disabled"}
                </span>
              </div>
            ))}
          </div>
        </div>

        <div className="space-y-6">
          {/* Logout Token Preview */}
          <div className="bg-gray-900 rounded-xl p-6">
            <h2 className="text-lg font-semibold mb-3">Logout Token Preview</h2>
            <div className="bg-gray-800 rounded-lg p-4 overflow-x-auto">
              <pre className="text-xs font-mono text-gray-300 whitespace-pre-wrap">{JSON.stringify(data?.logout_token_preview ?? {}, null, 2)}</pre>
            </div>
          </div>

          {/* Test Logout */}
          <div className="bg-gray-900 rounded-xl p-6">
            <h2 className="text-lg font-semibold mb-3">Test Backchannel Logout</h2>
            <p className="text-sm text-gray-400 mb-3">Send a test logout notification to configured clients.</p>
            <button
              onClick={() => testLogout()}
              className="px-4 py-2 bg-green-600 hover:bg-green-700 rounded-lg text-sm font-medium transition"
            >
              Send Test Logout
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
