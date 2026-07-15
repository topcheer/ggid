"use client";

import { useIdpDiscoveryConfig } from "@ggid/sdk-react";
import { Globe, Search, RefreshCw } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

export default function IdpDiscoveryConfigPage() {
  const t = useTranslations();

  const { data, loading, error, refresh, testDiscovery } = useIdpDiscoveryConfig();

  if (loading) return <div className="p-8 text-gray-400">Loading IdP discovery config...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">IdP Discovery Configuration</h1>
          <p className="text-sm text-gray-400 mt-1">Configure identity provider discovery methods</p>
        </div>
        <button onClick={refresh} className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition">Refresh</button>
      </div>

      {/* Discovery Methods */}
      <div className="bg-gray-900 rounded-xl p-6 mb-6">
        <h2 className="text-sm font-semibold mb-4">Discovery Methods</h2>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          {(data?.discovery_methods ?? []).map((m) => (
            <div key={m.method} className="bg-gray-800 rounded-lg p-4 border border-gray-700">
              <div className="flex items-center gap-2 mb-2">
                <Globe className="w-4 h-4 text-blue-400" />
                <h3 className="text-sm font-medium">{m.method}</h3>
              </div>
              <p className="text-xs text-gray-400 mb-2">{m.description}</p>
              <span className={"text-xs px-2 py-0.5 rounded " + (m.enabled ? "bg-green-900 text-green-300" : "bg-gray-700 text-gray-400")}>
                {m.enabled ? "Enabled" : "Disabled"}
              </span>
            </div>
          ))}
        </div>
      </div>

      {/* Email Domain Rules */}
      <div className="bg-gray-900 rounded-xl p-6 mb-6">
        <h2 className="text-sm font-semibold mb-4">Email Domain → IdP Mapping</h2>
        <div className="space-y-2">
          {(data?.email_domain_rules ?? []).map((r) => (
            <div key={r.domain} className="flex items-center gap-3 bg-gray-800 rounded-lg p-3">
              <span className="text-sm font-mono text-blue-400">*@{r.domain}</span>
              <span className="text-gray-600">{" -> "}</span>
              <span className="text-sm text-gray-300">{r.provider_name}</span>
              <span className="text-xs text-gray-500 ml-auto">{r.priority}</span>
            </div>
          ))}
        </div>
      </div>

      {/* Fallback Policy & Test */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-sm font-semibold mb-3">Fallback Policy</h2>
          <p className="text-sm text-gray-300">{data?.fallback_policy ?? "Show login form"}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-sm font-semibold mb-3">Test Discovery</h2>
          <button onClick={() => testDiscovery("test@corp.com")} className="flex items-center gap-2 px-4 py-2 bg-gray-700 hover:bg-gray-600 rounded-lg text-sm font-medium transition">
            <Search className="w-4 h-4" /> Test with sample email
          </button>
        </div>
      </div>

      {/* Discovery Log */}
      {data?.discovery_log && data.discovery_log.length > 0 && (
        <div className="bg-gray-900 rounded-xl p-6 mt-6">
          <h2 className="text-sm font-semibold mb-3">Recent Discovery Log</h2>
          <div className="space-y-1">
            {data.discovery_log.map((log) => (
              <div key={log.id} className="flex items-center gap-2 bg-gray-800 rounded p-2 text-xs">
                <span className="text-gray-500">{log.timestamp}</span>
                <span className="font-mono text-blue-400">{log.email}</span>
                <span className="text-gray-600">{" -> "}</span>
                <span className="text-gray-300">{log.provider}</span>
                <span className={"ml-auto px-1.5 py-0.5 rounded " + (log.result === "found" ? "bg-green-900 text-green-300" : "bg-yellow-900 text-yellow-300")}>{log.result}</span>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
