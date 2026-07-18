"use client";

import { useTorVpnDetect } from "@ggid/sdk-react";
import { Globe, Network, Shield, AlertTriangle, Eye } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

export default function TorVpnDetectPage() {
  const t = useTranslations();
  const { data, loading, error, refresh } = useTorVpnDetect();

  if (loading) return <div className="p-8 text-gray-400">Loading TOR/VPN detection...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">TOR / VPN Detection</h1>
          <p className="text-sm text-gray-400 mt-1">Detect connections from TOR exit nodes, VPNs, and proxies</p>
        </div>
        <button onClick={refresh} className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition">Refresh</button>
      </div>

      {/* Summary */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
        <div className="bg-gray-900 rounded-xl p-4">
          <Network className="w-5 h-5 text-red-400 mb-1" />
          <p className="text-xs text-gray-400">Detected Connections</p>
          <p className="text-xl font-bold">{data?.detected_connections?.length ?? 0}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <Shield className="w-5 h-5 text-yellow-400 mb-1" />
          <p className="text-xs text-gray-400">Exit Nodes</p>
          <p className="text-xl font-bold">{data?.exit_node_list?.length ?? 0}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <AlertTriangle className="w-5 h-5 text-orange-400 mb-1" />
          <p className="text-xs text-gray-400">Auto-Challenge</p>
          <p className="text-sm font-bold">{data?.auto_challenge_enabled ? "Enabled" : "Disabled"}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <Globe className="w-5 h-5 text-blue-400 mb-1" />
          <p className="text-xs text-gray-400">Countries</p>
          <p className="text-xl font-bold">{data?.per_country_stats?.length ?? 0}</p>
        </div>
      </div>

      {/* Detected Connections */}
      <div className="bg-gray-900 rounded-xl p-6 mb-6">
        <h2 className="text-lg font-semibold mb-4">Detected Connections</h2>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-gray-800 text-gray-400">
                <th scope="col" className="text-left py-2 pr-3">IP Address</th>
                <th scope="col" className="text-left py-2 pr-3">Type</th>
                <th scope="col" className="text-left py-2 pr-3">Confidence</th>
                <th scope="col" className="text-left py-2 pr-3">User</th>
                <th scope="col" className="text-left py-2 pr-3">First Seen</th>
                <th scope="col" className="text-left py-2 pr-3">Action</th>
              </tr>
            </thead>
            <tbody>
              {(data?.detected_connections ?? []).map((c: any) => (
                <tr key={c.ip} className="border-b border-gray-800">
                  <td className="py-3 pr-3 font-mono text-xs text-blue-400">{c.ip}</td>
                  <td className="py-3 pr-3">
                    <span className={"text-xs px-2 py-0.5 rounded " + (
                      c.type === "tor" ? "bg-red-900 text-red-300" :
                      c.type === "vpn" ? "bg-yellow-900 text-yellow-300" :
                      "bg-orange-900 text-orange-300"
                    )}>
                      {c.type.toUpperCase()}
                    </span>
                  </td>
                  <td className="py-3 pr-3">
                    <div className="flex items-center gap-2">
                      <div className="w-10 h-1.5 bg-gray-700 rounded-full">
                        <div className={"h-full rounded-full " + (c.confidence > 0.8 ? "bg-red-500" : "bg-yellow-500")} style={{ width: (c.confidence * 100) + "%" }} />
                      </div>
                      <span className="text-xs">{(c.confidence * 100).toFixed(0)}%</span>
                    </div>
                  </td>
                  <td className="py-3 pr-3 text-xs text-gray-400">{c.user}</td>
                  <td className="py-3 pr-3 text-xs text-gray-400">{c.first_seen}</td>
                  <td className="py-3 pr-3">
                    <button className="text-xs text-red-400 hover:underline">Block IP</button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Per Country Stats */}
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-sm font-semibold flex items-center gap-2 mb-4">
            <Globe className="w-4 h-4 text-blue-400" />
            Connections by Country
          </h2>
          <div className="space-y-2">
            {(data?.per_country_stats ?? []).map((s: any) => (
              <div key={s.country} className="flex items-center gap-3">
                <span className="text-sm w-24">{s.country}</span>
                <div className="flex-1 h-2 bg-gray-800 rounded-full">
                  <div className="h-full bg-red-500 rounded-full" style={{ width: (s.connections / Math.max(...(data?.per_country_stats?.map((x: any) => x.connections) ?? [1]))) * 100 + "%" }} />
                </div>
                <span className="text-xs text-gray-400 w-8 text-right">{s.connections}</span>
              </div>
            ))}
          </div>
        </div>

        {/* Blocklist Rules + Exit Nodes */}
        <div className="space-y-6">
          <div className="bg-gray-900 rounded-xl p-6">
            <h2 className="text-sm font-semibold flex items-center gap-2 mb-3">
              <Shield className="w-4 h-4 text-yellow-400" />
              Blocklist Rules
            </h2>
            <div className="space-y-2">
              {(data?.blocklist_rules ?? []).map((r: any) => (
                <div key={r.rule_name} className="flex items-center justify-between bg-gray-800 rounded-lg p-3">
                  <span className="text-sm">{r.rule_name}</span>
                  <span className={"text-xs px-2 py-0.5 rounded " + (r.enabled ? "bg-green-900 text-green-300" : "bg-gray-700 text-gray-400")}>
                    {r.enabled ? "On" : "Off"}
                  </span>
                </div>
              ))}
            </div>
          </div>

          <div className="bg-gray-900 rounded-xl p-6">
            <h2 className="text-sm font-semibold flex items-center gap-2 mb-3">
              <Eye className="w-4 h-4 text-red-400" />
              Active Exit Nodes (sample)
            </h2>
            <div className="flex flex-wrap gap-1">
              {(data?.exit_node_list ?? []).slice(0, 8).map((node: any) => (
                <span key={node} className="text-xs font-mono px-1.5 py-0.5 bg-gray-800 rounded text-gray-400">{node}</span>
              ))}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
