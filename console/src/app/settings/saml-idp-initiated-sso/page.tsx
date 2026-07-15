"use client";

import { useSamlIdpInitiatedSSO } from "@ggid/sdk-react";
import { Globe, Shield, Link2, AlertTriangle, Play, CheckCircle, Settings } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

export default function SamlIdpInitiatedSSOPage() {
  const t = useTranslations();
  const { data, loading, error, refresh, testSso } = useSamlIdpInitiatedSSO();

  if (loading) return <div className="p-8 text-gray-400">Loading IdP-initiated SSO config...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">IdP-Initiated SSO</h1>
          <p className="text-sm text-gray-400 mt-1">Configure Identity Provider-initiated Single Sign-On</p>
        </div>
        <button
          onClick={refresh}
          className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition"
        >
          Refresh
        </button>
      </div>

      {/* Security Warnings */}
      {(data?.security_warnings ?? []).length > 0 && (
        <div className="bg-yellow-950 border border-yellow-700 rounded-xl p-4 mb-6">
          <h2 className="text-sm font-semibold flex items-center gap-2 mb-2 text-yellow-300">
            <AlertTriangle className="w-4 h-4" />
            Security Warnings
          </h2>
          <ul className="space-y-1">
            {(data?.security_warnings ?? []).map((w, i) => (
              <li key={i} className="text-xs text-yellow-200">- {w}</li>
            ))}
          </ul>
        </div>
      )}

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Allowed IdPs */}
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
            <Globe className="w-5 h-5 text-blue-400" />
            Allowed Identity Providers
          </h2>
          <div className="space-y-2">
            {(data?.allowed_idps ?? []).map((idp) => (
              <div key={idp.entity_id} className="bg-gray-800 rounded-lg p-3">
                <div className="flex items-center justify-between mb-1">
                  <div>
                    <p className="text-sm font-medium">{idp.provider_name}</p>
                    <p className="text-xs text-gray-400 font-mono">{idp.entity_id}</p>
                  </div>
                  <span
                    className={"text-xs px-2 py-0.5 rounded " + (
                      idp.status === "active" ? "bg-green-900 text-green-300" :
                      idp.status === "pending" ? "bg-yellow-900 text-yellow-300" :
                      "bg-gray-700 text-gray-400"
                    )}
                  >
                    {idp.status}
                  </span>
                </div>
                {idp.idp_initiated_enabled && (
                  <div className="flex items-center gap-1 mt-1 text-xs text-green-400">
                    <CheckCircle className="w-3 h-3" />
                    IdP-Initiated enabled
                  </div>
                )}
              </div>
            ))}
          </div>
        </div>

        <div className="space-y-6">
          {/* Relay State + SSO URL */}
          <div className="bg-gray-900 rounded-xl p-6">
            <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
              <Link2 className="w-5 h-5 text-purple-400" />
              Relay State & SSO URL
            </h2>
            <div className="space-y-3">
              <div className="bg-gray-800 rounded-lg p-3">
                <p className="text-xs text-gray-400 mb-1">Relay State Config</p>
                <p className="text-sm font-medium">{data?.relay_state_config ?? "Default"}</p>
              </div>
              <div className="bg-gray-800 rounded-lg p-3">
                <p className="text-xs text-gray-400 mb-1">SSO URL Preview</p>
                <p className="text-sm font-mono text-blue-400 break-all">{data?.sso_url_preview ?? "N/A"}</p>
              </div>
            </div>
          </div>

          {/* Session Bridge */}
          <div className="bg-gray-900 rounded-xl p-6">
            <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
              <Settings className="w-5 h-5 text-blue-400" />
              Session Bridge
            </h2>
            <div className="space-y-3">
              <div className="flex items-center justify-between bg-gray-800 rounded-lg p-3">
                <span className="text-sm text-gray-300">Create Local Session</span>
                <span
                  className={"text-xs px-2 py-0.5 rounded " + (
                    data?.session_bridge?.create_local_session ? "bg-green-900 text-green-300" : "bg-gray-700 text-gray-400"
                  )}
                >
                  {data?.session_bridge?.create_local_session ? "Enabled" : "Disabled"}
                </span>
              </div>
              <div className="bg-gray-800 rounded-lg p-3">
                <p className="text-xs text-gray-400 mb-2">Mapped Attributes</p>
                <div className="flex flex-wrap gap-1">
                  {(data?.session_bridge?.map_attributes ?? []).map((attr) => (
                    <span key={attr} className="text-xs px-2 py-0.5 rounded bg-blue-900 text-blue-300">{attr}</span>
                  ))}
                </div>
              </div>
            </div>
          </div>

          {/* Test SSO */}
          <div className="bg-gray-900 rounded-xl p-6">
            <h2 className="text-lg font-semibold flex items-center gap-2 mb-3">
              <Shield className="w-5 h-5 text-green-400" />
              Test SSO Flow
            </h2>
            <p className="text-sm text-gray-400 mb-3">Initiate a test IdP-initiated SSO login.</p>
            <button
              onClick={() => testSso()}
              className="flex items-center gap-2 px-4 py-2 bg-green-600 hover:bg-green-700 rounded-lg text-sm font-medium transition"
            >
              <Play className="w-4 h-4" />
              Test SSO
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
