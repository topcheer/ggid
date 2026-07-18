"use client";

import { useOAuthJwksManagement } from "@ggid/sdk-react";
import { Key, RotateCw, CheckCircle, XCircle, Clock, Plus, Activity } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

export default function OAuthJwksManagementPage() {
  const t = useTranslations();

  const { data, loading, error, refresh, rotateKey, testEndpoint } = useOAuthJwksManagement();

  if (loading) return <div className="p-8 text-gray-400">Loading JWKS management...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">JWKS Management</h1>
          <p className="text-sm text-gray-400 mt-1">Manage JSON Web Key Set for OAuth/OIDC token signing</p>
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={() => rotateKey()}
            className="flex items-center gap-2 px-4 py-2 bg-green-600 hover:bg-green-700 rounded-lg text-sm font-medium transition"
          >
            <Plus className="w-4 h-4" />
            Rotate Key
          </button>
          <button
            onClick={refresh}
            className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition"
          >
            Refresh
          </button>
        </div>
      </div>

      {/* Top Metrics */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-blue-400">
            <Key className="w-4 h-4" />
            <span className="text-xs text-gray-400">Active Keys</span>
          </div>
          <p className="text-2xl font-bold">{data?.active_keys?.filter((k) => k.status === "active").length ?? 0}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-green-400">
            <Clock className="w-4 h-4" />
            <span className="text-xs text-gray-400">Auto-Rotation</span>
          </div>
          <p className="text-lg font-bold">{data?.auto_rotation_interval_days ?? 0} days</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-purple-400">
            <Activity className="w-4 h-4" />
            <span className="text-xs text-gray-400">KID Strategy</span>
          </div>
          <p className="text-sm font-mono text-purple-300">{data?.kid_strategy ?? "x5t#S256"}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-yellow-400">
            <Activity className="w-4 h-4" />
            <span className="text-xs text-gray-400">JWKS Health</span>
          </div>
          <p className={"text-lg font-bold " + (data?.jwks_uri_health?.healthy ? "text-green-400" : "text-red-400")}>
            {data?.jwks_uri_health?.healthy ? "Healthy" : "Unhealthy"}
          </p>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Active Keys */}
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
            <Key className="w-5 h-5 text-blue-400" />
            Active Keys
          </h2>
          <div className="space-y-2">
            {(data?.active_keys ?? []).map((key) => (
              <div key={key.kid} className="bg-gray-800 rounded-lg p-3">
                <div className="flex items-center justify-between mb-2">
                  <code className="text-xs font-mono text-blue-400">{key.kid}</code>
                  <span
                    className={"text-xs px-2 py-0.5 rounded " + (
                      key.status === "active" ? "bg-green-900 text-green-300" :
                      key.status === "rotated" ? "bg-yellow-900 text-yellow-300" :
                      "bg-red-900 text-red-300"
                    )}
                  >
                    {key.status}
                  </span>
                </div>
                <div className="grid grid-cols-3 gap-2 text-xs">
                  <div>
                    <span className="text-gray-500">Alg: </span>
                    <span className="font-medium">{key.alg}</span>
                  </div>
                  <div>
                    <span className="text-gray-500">Kty: </span>
                    <span className="font-medium">{key.kty}</span>
                  </div>
                  <div>
                    <span className="text-gray-500">Created: </span>
                    <span>{key.created_at}</span>
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>

        <div className="space-y-6">
          {/* Rotation History */}
          <div className="bg-gray-900 rounded-xl p-6">
            <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
              <RotateCw className="w-5 h-5 text-blue-400" />
              Key Rotation History
            </h2>
            <div className="space-y-2 max-h-48 overflow-y-auto">
              {(data?.key_rotation_history ?? []).map((r: any, i: number) => (
                <div key={i} className="flex items-center gap-3 bg-gray-800 rounded-lg p-3">
                  {r.success ? (
                    <CheckCircle className="w-4 h-4 text-green-400 flex-shrink-0" />
                  ) : (
                    <XCircle className="w-4 h-4 text-red-400 flex-shrink-0" />
                  )}
                  <div className="flex-1 min-w-0">
                    <p className="text-sm font-medium truncate">{r.old_kid}{" -> "}{r.new_kid}</p>
                    <p className="text-xs text-gray-400">{r.timestamp} by {r.triggered_by}</p>
                  </div>
                </div>
              ))}
            </div>
          </div>

          {/* JWKS URI Health + Test */}
          <div className="bg-gray-900 rounded-xl p-6">
            <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
              <Activity className="w-5 h-5 text-green-400" />
              JWKS URI Health Check
            </h2>
            <div className="space-y-2 mb-4">
              <div className="flex items-center justify-between bg-gray-800 rounded-lg p-3">
                <span className="text-sm text-gray-300">URI</span>
                <code className="text-xs font-mono text-blue-400">{data?.jwks_uri_health?.uri ?? "N/A"}</code>
              </div>
              <div className="flex items-center justify-between bg-gray-800 rounded-lg p-3">
                <span className="text-sm text-gray-300">Response Time</span>
                <span className="text-sm font-medium">{data?.jwks_uri_health?.response_time_ms ?? 0}ms</span>
              </div>
              <div className="flex items-center justify-between bg-gray-800 rounded-lg p-3">
                <span className="text-sm text-gray-300">Cache Hit Rate</span>
                <span className="text-sm font-medium">{data?.jwks_uri_health?.cache_hit_rate ?? 0}%</span>
              </div>
            </div>
            <button
              onClick={() => testEndpoint()}
              className="flex items-center gap-2 px-4 py-2 bg-green-600 hover:bg-green-700 rounded-lg text-sm font-medium transition"
            >
              Test Endpoint
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
