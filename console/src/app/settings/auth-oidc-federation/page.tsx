"use client";

import { useAuthOidcFederation } from "@ggid/sdk-react";
import { Globe, ShieldCheck, Link2, FileText, Search, CheckCircle, AlertTriangle } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

export default function AuthOidcFederationPage() {
  const t = useTranslations();

  const { data, loading, error, refresh } = useAuthOidcFederation();

  if (loading) return <div className="p-8 text-gray-400">Loading OIDC federation...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">OIDC Federation</h1>
          <p className="text-sm text-gray-400 mt-1">Manage trust anchors, federated providers, and trust chains</p>
        </div>
        <button
          onClick={refresh}
          className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition"
        >
          Refresh
        </button>
      </div>

      {/* Trust Resolution Status + Auto Discovery */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-6">
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1">
            {data?.trust_resolution_status === "healthy" ? (
              <CheckCircle className="w-4 h-4 text-green-400" />
            ) : (
              <AlertTriangle className="w-4 h-4 text-yellow-400" />
            )}
            <span className="text-xs text-gray-400">Trust Resolution</span>
          </div>
          <p className="text-lg font-bold capitalize">{data?.trust_resolution_status ?? "unknown"}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-blue-400">
            <Globe className="w-4 h-4" />
            <span className="text-xs text-gray-400">Trust Anchors</span>
          </div>
          <p className="text-lg font-bold">{data?.trust_anchors?.length ?? 0}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-purple-400">
            <Link2 className="w-4 h-4" />
            <span className="text-xs text-gray-400">Federated Providers</span>
          </div>
          <p className="text-lg font-bold">{data?.federated_providers?.length ?? 0}</p>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Trust Anchors */}
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
            <ShieldCheck className="w-5 h-5 text-blue-400" />
            Trust Anchors
          </h2>
          <div className="space-y-2">
            {(data?.trust_anchors ?? []).map((anchor: any) => (
              <div key={anchor.issuer} className="bg-gray-800 rounded-lg p-3">
                <div className="flex items-center justify-between mb-1">
                  <p className="text-sm font-medium">{anchor.issuer}</p>
                  <span
                    className={"text-xs px-2 py-0.5 rounded " + (
                      anchor.trust_mark_valid
                        ? "bg-green-900 text-green-300"
                        : "bg-red-900 text-red-300"
                    )}
                  >
                    {anchor.trust_mark_valid ? "Valid" : "Invalid"}
                  </span>
                </div>
                <p className="text-xs text-gray-400 font-mono break-all">{anchor.jwks_uri}</p>
              </div>
            ))}
          </div>
        </div>

        {/* Federated Providers */}
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
            <Link2 className="w-5 h-5 text-purple-400" />
            Federated Providers
          </h2>
          <div className="space-y-2">
            {(data?.federated_providers ?? []).map((p: any) => (
              <div key={p.entity_id} className="bg-gray-800 rounded-lg p-3">
                <div className="flex items-center justify-between mb-1">
                  <div>
                    <p className="text-sm font-medium">{p.entity_id}</p>
                    <p className="text-xs text-gray-400">{p.organization}</p>
                  </div>
                  <span
                    className={"text-xs px-2 py-0.5 rounded " + (
                      p.status === "active" ? "bg-green-900 text-green-300" :
                      p.status === "pending" ? "bg-yellow-900 text-yellow-300" :
                      "bg-gray-700 text-gray-400"
                    )}
                  >
                    {p.status}
                  </span>
                </div>
                <div className="flex items-center gap-2 mt-1">
                  <span className="text-xs text-gray-500">Role:</span>
                  <span className="text-xs text-gray-300">{p.role}</span>
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>

      {/* Trust Chain Visual + Entity Statement */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 mt-6">
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-lg font-semibold mb-4">Trust Chain</h2>
          <div className="space-y-2">
            {(data?.trust_chain ?? []).map((node: any, i: number) => (
              <div key={i} className="flex items-center gap-3">
                <div className={"w-8 h-8 rounded-lg flex items-center justify-center text-xs font-bold " + (
                  node.verified ? "bg-green-900 text-green-300" : "bg-yellow-900 text-yellow-300"
                )}>
                  {i + 1}
                </div>
                <div className="flex-1">
                  <p className="text-sm font-medium">{node.entity}</p>
                  <p className="text-xs text-gray-400">{node.metadata_type}</p>
                </div>
                {node.verified ? (
                  <CheckCircle className="w-4 h-4 text-green-400" />
                ) : (
                  <AlertTriangle className="w-4 h-4 text-yellow-400" />
                )}
                {i < (data?.trust_chain ?? []).length - 1 && (
                  <div className="absolute" style={{ left: "19px" }} />
                )}
              </div>
            ))}
          </div>
        </div>

        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
            <FileText className="w-5 h-5 text-blue-400" />
            Entity Statement Viewer
          </h2>
          <div className="bg-gray-800 rounded-lg p-4 overflow-x-auto">
            <pre className="text-xs font-mono text-gray-300 whitespace-pre-wrap">{JSON.stringify(data?.entity_statement ?? {}, null, 2)}</pre>
          </div>
          <div className="mt-4 flex items-center gap-2">
            <button className="flex items-center gap-1 px-3 py-2 bg-gray-700 hover:bg-gray-600 rounded-lg text-sm font-medium transition">
              <Search className="w-4 h-4" />
              Auto-Discovery
            </button>
            <span className="text-xs text-gray-400">Last discovery: {data?.last_auto_discovery ?? "N/A"}</span>
          </div>
        </div>
      </div>
    </div>
  );
}
