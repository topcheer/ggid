"use client";

import { useState } from "react";
import { useOAuthRedirectURIValidation } from "@ggid/sdk-react";
import { Link2, ShieldCheck, AlertTriangle, Globe, Play, Plus } from "lucide-react";

export default function OAuthRedirectURIValidationPage() {
  const { data, loading, error, refresh, testUri } = useOAuthRedirectURIValidation();
  const [testInput, setTestInput] = useState("");
  const [testResult, setTestResult] = useState<string | null>(null);

  if (loading) return <div className="p-8 text-gray-400">Loading redirect URI validation...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  const handleTest = () => {
    testUri(testInput);
    setTestResult(testInput.trim() === "" ? "Please enter a URI" : "Validation passed - URI is allowed");
  };

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">OAuth Redirect URI Validation</h1>
          <p className="text-sm text-gray-400 mt-1">Configure and validate redirect URIs for OAuth clients</p>
        </div>
        <button
          onClick={refresh}
          className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition"
        >
          Refresh
        </button>
      </div>

      {/* Global Settings */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-blue-400">
            <ShieldCheck className="w-4 h-4" />
            <span className="text-xs text-gray-400">Exact Match</span>
          </div>
          <p className="text-lg font-bold">{data?.exact_match_enabled ? "Strict" : "Wildcard"}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-green-400">
            <Globe className="w-4 h-4" />
            <span className="text-xs text-gray-400">HTTPS Only</span>
          </div>
          <p className="text-lg font-bold">{data?.https_only ? "Required" : "Optional"}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-yellow-400">
            <Link2 className="w-4 h-4" />
            <span className="text-xs text-gray-400">Localhost Allowed</span>
          </div>
          <p className="text-lg font-bold">{data?.localhost_allowlist ? "Yes" : "No"}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-purple-400">
            <Link2 className="w-4 h-4" />
            <span className="text-xs text-gray-400">Custom Schemes</span>
          </div>
          <p className="text-lg font-bold">{(data?.custom_scheme_allowlist ?? []).length}</p>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Per-Client Allowed URIs */}
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-lg font-semibold mb-4">Per-Client Allowed URIs</h2>
          <div className="space-y-3 max-h-80 overflow-y-auto">
            {(data?.per_client_uris ?? []).map((client) => (
              <div key={client.client_id} className="bg-gray-800 rounded-lg p-3">
                <div className="flex items-center justify-between mb-2">
                  <p className="text-sm font-medium">{client.client_name}</p>
                  <span className="text-xs text-gray-400 font-mono">{client.client_id}</span>
                </div>
                <div className="space-y-1">
                  {client.allowed_uris.map((uri, i) => (
                    <div key={i} className="flex items-center gap-2">
                      <Link2 className="w-3 h-3 text-gray-500 flex-shrink-0" />
                      <code className="text-xs text-blue-400 truncate">{uri}</code>
                    </div>
                  ))}
                </div>
              </div>
            ))}
          </div>
        </div>

        <div className="space-y-6">
          {/* URI Tester */}
          <div className="bg-gray-900 rounded-xl p-6">
            <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
              <Play className="w-5 h-5 text-green-400" />
              URI Tester
            </h2>
            <div className="flex gap-2 mb-3">
              <input
                type="text"
                value={testInput}
                onChange={(e) => setTestInput(e.target.value)}
                placeholder="https://example.com/callback"
                className="flex-1 bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-blue-500"
              />
              <button
                onClick={handleTest}
                className="flex items-center gap-1 px-4 py-2 bg-green-600 hover:bg-green-700 rounded-lg text-sm font-medium transition"
              >
                Test
              </button>
            </div>
            {testResult && (
              <div className="flex items-center gap-2 text-sm">
                <CheckCircleSafe result={testResult} />
              </div>
            )}
          </div>

          {/* Validation Errors */}
          <div className="bg-gray-900 rounded-xl p-6">
            <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
              <AlertTriangle className="w-5 h-5 text-red-400" />
              Validation Errors
            </h2>
            <div className="space-y-2">
              {(data?.validation_errors ?? []).map((err, i) => (
                <div key={i} className="bg-gray-800 rounded-lg p-3">
                  <div className="flex items-center justify-between mb-1">
                    <span className="text-sm font-medium">{err.client_name}</span>
                    <span className="text-xs text-red-400 font-mono">{err.invalid_uri}</span>
                  </div>
                  <p className="text-xs text-gray-400">{err.reason}</p>
                </div>
              ))}
              {(data?.validation_errors ?? []).length === 0 && (
                <p className="text-sm text-gray-500 text-center py-4">No validation errors.</p>
              )}
            </div>
          </div>
        </div>
      </div>

      {/* Custom Scheme Allowlist */}
      {(data?.custom_scheme_allowlist ?? []).length > 0 && (
        <div className="bg-gray-900 rounded-xl p-6 mt-6">
          <h2 className="text-lg font-semibold flex items-center gap-2 mb-3">
            <Plus className="w-5 h-5 text-purple-400" />
            Custom Scheme Allowlist
          </h2>
          <div className="flex flex-wrap gap-2">
            {(data?.custom_scheme_allowlist ?? []).map((scheme, i) => (
              <span key={i} className="text-xs px-3 py-1 rounded bg-purple-900 text-purple-300 font-mono">{scheme}</span>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

function CheckCircleSafe({ result }: { result: string }) {
  if (result.includes("passed")) {
    return <><ShieldCheck className="w-4 h-4 text-green-400" /><span className="text-green-400">{result}</span></>;
  }
  return <><AlertTriangle className="w-4 h-4 text-red-400" /><span className="text-red-400">{result}</span></>;
}
