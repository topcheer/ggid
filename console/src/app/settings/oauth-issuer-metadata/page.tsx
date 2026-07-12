"use client";

import { useOAuthIssuerMetadata } from "@ggid/sdk-react";
import { Globe, FileJson, CheckCircle, XCircle, RefreshCw } from "lucide-react";

export default function OAuthIssuerMetadataPage() {
  const { data, loading, error, refresh } = useOAuthIssuerMetadata();

  if (loading) return <div className="p-8 text-gray-400">Loading issuer metadata...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">Issuer Metadata</h1>
          <p className="text-sm text-gray-400 mt-1">OAuth/OIDC well-known discovery document configuration</p>
        </div>
        <button
          onClick={refresh}
          className="flex items-center gap-2 px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition"
        >
          <RefreshCw className="w-4 h-4" />
          Refresh
        </button>
      </div>

      {/* Issuer URL */}
      <div className="bg-gray-900 rounded-xl p-6 mb-6">
        <div className="flex items-center gap-3">
          <Globe className="w-5 h-5 text-blue-400" />
          <div>
            <p className="text-xs text-gray-400">Issuer URL</p>
            <p className="text-sm font-mono">{data?.issuer_url}</p>
          </div>
          <span className="text-xs text-gray-500 ml-auto">{data?.well_known_path}</span>
        </div>
      </div>

      {/* Supported Capabilities */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-6">
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-sm font-semibold mb-4">Supported Response Types</h2>
          <div className="flex flex-wrap gap-2">
            {(data?.supported_response_types ?? []).map((rt) => (
              <span key={rt} className="text-xs px-3 py-1.5 bg-gray-800 rounded-lg border border-gray-700 font-mono">{rt}</span>
            ))}
          </div>
        </div>
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-sm font-semibold mb-4">Supported Subject Types</h2>
          <div className="flex flex-wrap gap-2">
            {(data?.supported_subject_types ?? []).map((st) => (
              <span key={st} className="text-xs px-3 py-1.5 bg-gray-800 rounded-lg border border-gray-700 font-mono">{st}</span>
            ))}
          </div>
        </div>
      </div>

      {/* Feature Toggles */}
      <div className="bg-gray-900 rounded-xl p-6 mb-6">
        <h2 className="text-sm font-semibold mb-4">Feature Support</h2>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
          {[
            { label: "Claim Types Supported", value: data?.claim_types_supported ?? [] },
            { label: "Request Parameter", value: data?.request_parameter_supported },
            { label: "Request URI Parameter", value: data?.request_uri_parameter_supported },
            { label: "Require Request URI Registration", value: data?.require_request_uri_registration },
          ].map((item) => (
            <div key={item.label} className="bg-gray-800 rounded-lg p-3">
              <p className="text-xs text-gray-400 mb-1">{item.label}</p>
              {Array.isArray(item.value) ? (
                <div className="flex flex-wrap gap-1">
                  {(item.value as string[]).map((v) => (
                    <span key={v} className="text-xs px-1.5 py-0.5 bg-gray-700 rounded">{v}</span>
                  ))}
                </div>
              ) : (
                <div className="flex items-center gap-1">
                  {item.value ? <CheckCircle className="w-3 h-3 text-green-400" /> : <XCircle className="w-3 h-3 text-gray-500" />}
                  <span className={"text-xs " + (item.value ? "text-green-400" : "text-gray-500")}>{item.value ? "Supported" : "Not Supported"}</span>
                </div>
              )}
            </div>
          ))}
        </div>
      </div>

      {/* Well-Known Preview */}
      <div className="bg-gray-900 rounded-xl p-6">
        <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
          <FileJson className="w-5 h-5 text-purple-400" />
          Well-Known Preview
        </h2>
        <div className="bg-gray-800 rounded-lg p-4 overflow-x-auto">
          <pre className="text-xs font-mono text-gray-300 whitespace-pre-wrap">{JSON.stringify(data?.well_known_preview ?? {}, null, 2)}</pre>
        </div>
      </div>
    </div>
  );
}
