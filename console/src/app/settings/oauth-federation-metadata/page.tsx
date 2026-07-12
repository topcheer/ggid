"use client";

import { useOAuthFederationMetadata } from "@ggid/sdk-react";
import { Download, RefreshCw, FileCode } from "lucide-react";

export default function OAuthFederationMetadataPage() {
  const { data, loading, error, refresh } = useOAuthFederationMetadata();

  if (loading) return <div className="p-8 text-gray-400">Loading federation metadata...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">Federation Metadata</h1>
          <p className="text-sm text-gray-400 mt-1">Manage federated entity metadata and trust</p>
        </div>
        <div className="flex gap-2">
          <button className="flex items-center gap-1 px-4 py-2 bg-gray-700 hover:bg-gray-600 rounded-lg text-sm font-medium transition">
            <Download className="w-4 h-4" /> Import
          </button>
          <button onClick={refresh} className="flex items-center gap-1 px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition">
            <RefreshCw className="w-4 h-4" /> Refresh All
          </button>
        </div>
      </div>

      {/* Federated Entities */}
      <div className="bg-gray-900 rounded-xl p-6">
        <h2 className="text-sm font-semibold mb-4">Federated Entities</h2>
        <div className="space-y-3">
          {(data?.federated_entities ?? []).map((e) => (
            <div key={e.entity_id} className="bg-gray-800 rounded-lg p-4">
              <div className="flex items-start justify-between mb-2">
                <div className="flex items-center gap-3">
                  <FileCode className="w-5 h-5 text-purple-400" />
                  <div>
                    <h3 className="text-sm font-semibold font-mono">{e.entity_id}</h3>
                    <p className="text-xs text-gray-400">Role: {e.role}</p>
                  </div>
                </div>
                <span className={"text-xs px-2 py-0.5 rounded " + (
                  e.trust_status === "trusted" ? "bg-green-900 text-green-300" :
                  e.trust_status === "pending" ? "bg-yellow-900 text-yellow-300" :
                  "bg-red-900 text-red-300"
                )}>{e.trust_status}</span>
              </div>
              <div className="grid grid-cols-3 gap-4 mt-3">
                <div>
                  <p className="text-xs text-gray-500">Metadata URL</p>
                  <p className="text-xs font-mono text-blue-400 truncate">{e.metadata_url}</p>
                </div>
                <div>
                  <p className="text-xs text-gray-500">Last Refresh</p>
                  <p className="text-xs text-gray-300">{e.last_refresh}</p>
                </div>
                <div>
                  <p className="text-xs text-gray-500">Entity Categories</p>
                  <div className="flex flex-wrap gap-1">
                    {e.entity_categories.map((cat) => (
                      <span key={cat} className="text-xs px-1.5 py-0.5 bg-gray-700 rounded text-gray-400">{cat}</span>
                    ))}
                  </div>
                </div>
              </div>
            </div>
          ))}
        </div>
      </div>

      {/* Auto-Refresh Schedule */}
      <div className="bg-gray-900 rounded-xl p-6 mt-6">
        <h2 className="text-sm font-semibold mb-3">Auto-Refresh Schedule</h2>
        <div className="flex items-center gap-4">
          <div>
            <p className="text-xs text-gray-500">Frequency</p>
            <p className="text-sm">{data?.auto_refresh_schedule ?? "Every 24h"}</p>
          </div>
          <div>
            <p className="text-xs text-gray-500">Next Refresh</p>
            <p className="text-sm text-blue-400">{data?.next_refresh ?? "--"}</p>
          </div>
        </div>
      </div>
    </div>
  );
}
