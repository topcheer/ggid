"use client";

import { useState } from "react";
import { useOAuthResourceIndicators } from "@ggid/sdk-react";
import { useTranslations } from "@/lib/i18n";
import { Globe, Shield, AlertTriangle, TestTube } from "lucide-react";

export default function OAuthResourceIndicatorsPage() {
  const t = useTranslations();
  const { data, loading, error, refresh, testResource } = useOAuthResourceIndicators();
  const [testInput, setTestInput] = useState("");
  const [testResult, setTestResult] = useState<string | null>(null);

  if (loading) return <div className="p-8 text-gray-400">Loading resource indicators...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">Resource Indicators</h1>
          <p className="text-sm text-gray-400 mt-1">Configure resource access patterns and audience validation</p>
        </div>
        <button
          onClick={refresh}
          className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition"
        >
          Refresh
        </button>
      </div>

      {/* Config Summary */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-6">
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-blue-400">
            <Globe className="w-4 h-4" />
            <span className="text-xs text-gray-400">Indicator Required</span>
          </div>
          <p className="text-lg font-bold">{data?.resource_indicator_required ? "Yes" : "No"}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-green-400">
            <Shield className="w-4 h-4" />
            <span className="text-xs text-gray-400">Clients Configured</span>
          </div>
          <p className="text-lg font-bold">{data?.per_client_patterns?.length ?? 0}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-1 text-red-400">
            <AlertTriangle className="w-4 h-4" />
            <span className="text-xs text-gray-400">Rejected (24h)</span>
          </div>
          <p className="text-lg font-bold">{data?.rejected_requests_log?.length ?? 0}</p>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Per-Client Patterns */}
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-lg font-semibold mb-4">Allowed Resource Patterns</h2>
          <div className="space-y-3">
            {(data?.per_client_patterns ?? []).map((client: any) => (
              <div key={client.client_id} className="bg-gray-800 rounded-lg p-3">
                <p className="text-sm font-mono text-blue-400 mb-2">{client.client_id}</p>
                <div className="space-y-1">
                  {client.patterns.map((p: any, i: number) => (
                    <div key={i} className="flex items-center gap-2">
                      <span
                        className={"text-xs px-2 py-0.5 rounded " + (
                          p.match_type === "exact" ? "bg-green-900 text-green-300" :
                          p.match_type === "wildcard" ? "bg-yellow-900 text-yellow-300" :
                          "bg-purple-900 text-purple-300"
                        )}
                      >
                        {p.match_type}
                      </span>
                      <code className="text-xs text-gray-300 font-mono">{p.pattern}</code>
                    </div>
                  ))}
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* Per-Scope Resource Restriction */}
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-lg font-semibold mb-4">Per-Scope Resource Restriction</h2>
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-gray-800 text-gray-400">
                  <th scope="col" className="text-left py-2 pr-3">Scope</th>
                  <th scope="col" className="text-left py-2 pr-3">Allowed Resources</th>
                  <th scope="col" className="text-left py-2 pr-3">Restricted</th>
                </tr>
              </thead>
              <tbody>
                {(data?.per_scope_restriction ?? []).map((s: any) => (
                  <tr key={s.scope} className="border-b border-gray-800">
                    <td className="py-2 pr-3 font-mono text-xs text-blue-400">{s.scope}</td>
                    <td className="py-2 pr-3 text-gray-300 text-xs">{s.allowed_resources.join(", ")}</td>
                    <td className="py-2 pr-3">
                      <span className={"text-xs " + (s.restricted ? "text-red-400" : "text-green-400")}>
                        {s.restricted ? "Yes" : "No"}
                      </span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      </div>

      {/* Resource Tester */}
      <div className="bg-gray-900 rounded-xl p-6 mt-6">
        <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
          <TestTube className="w-5 h-5 text-green-400" />
          Resource Indicator Tester
        </h2>
        <div className="flex items-center gap-2">
          <input
            type="text"
            placeholder="https://api.example.com/v1/users"
            value={testInput}
            onChange={(e) => setTestInput(e.target.value)}
            className="flex-1 bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm font-mono focus:outline-none focus:border-blue-500"
          />
          <button
            onClick={() => { const result = testResource(testInput); setTestResult(result ? "Allowed" : "Rejected"); }}
            className="px-4 py-2 bg-green-600 hover:bg-green-700 rounded-lg text-sm font-medium transition"
          >
            Test
          </button>
        </div>
        {testResult && (
          <div className="mt-3">
            <span
              className={"text-sm font-medium " + (testResult === "Allowed" ? "text-green-400" : "text-red-400")}
            >
              Result: {testResult}
            </span>
          </div>
        )}
      </div>

      {/* Rejected Requests Log */}
      {(data?.rejected_requests_log ?? []).length > 0 && (
        <div className="bg-gray-900 rounded-xl p-6 mt-6">
          <h2 className="text-lg font-semibold flex items-center gap-2 mb-4">
            <AlertTriangle className="w-5 h-5 text-red-400" />
            Rejected Requests Log
          </h2>
          <div className="space-y-2 max-h-48 overflow-y-auto">
            {(data?.rejected_requests_log ?? []).map((r: any, i: number) => (
              <div key={i} className="flex items-center gap-3 bg-gray-800 rounded-lg p-2">
                <AlertTriangle className="w-3 h-3 text-red-400 flex-shrink-0" />
                <div className="flex-1">
                  <p className="text-xs font-mono text-gray-300">{r.requested_resource}</p>
                  <p className="text-xs text-gray-500">Client: {r.client} - {r.reason}</p>
                </div>
                <span className="text-xs text-gray-500">{r.timestamp}</span>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
