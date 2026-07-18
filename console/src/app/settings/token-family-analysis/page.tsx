"use client";

import { useTokenFamilyAnalysis } from "@ggid/sdk-react";
import { useTranslations } from "@/lib/i18n";
import { GitBranch, AlertTriangle, Layers } from "lucide-react";

export default function TokenFamilyAnalysisPage() {
  const t = useTranslations();
  const { data, loading, error, refresh } = useTokenFamilyAnalysis();
  if (loading) return <div className="p-8 text-gray-400">Loading token families...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div><h1 className="text-2xl font-bold">Token Family Analysis</h1><p className="text-sm text-gray-400 mt-1">Track token derivation chains and reuse</p></div>
        <button onClick={refresh} className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition">Refresh</button>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-6">
        <div className="bg-gray-900 rounded-xl p-4"><Layers className="w-5 h-5 text-blue-400 mb-1" /><p className="text-xs text-gray-400">Active Families</p><p className="text-xl font-bold">{data?.families?.filter((f: any) => f.status === "active").length ?? 0}</p></div>
        <div className="bg-gray-900 rounded-xl p-4"><p className="text-xs text-gray-400">Total Child Tokens</p><p className="text-xl font-bold">{data?.families?.reduce((a, f) => a + f.child_count, 0) ?? 0}</p></div>
        <div className="bg-gray-900 rounded-xl p-4"><AlertTriangle className="w-5 h-5 text-red-400 mb-1" /><p className="text-xs text-gray-400">Reuse Alerts</p><p className="text-xl font-bold text-red-400">{data?.reuse_alerts?.length ?? 0}</p></div>
      </div>

      <div className="bg-gray-900 rounded-xl p-6 mb-6">
        <h2 className="text-sm font-semibold mb-4 flex items-center gap-2"><GitBranch className="w-4 h-4 text-blue-400" /> Token Families</h2>
        <div className="space-y-3">
          {(data?.families ?? []).map((f: any) => (
            <div key={f.family_id} className="bg-gray-800 rounded-lg p-4">
              <div className="flex items-center gap-3 mb-2">
                <span className="text-xs font-mono text-blue-400">{f.family_id}</span>
                <span className={"text-xs px-2 py-0.5 rounded " + (f.status === "active" ? "bg-green-900 text-green-300" : "bg-red-900 text-red-300")}>{f.status}</span>
                <span className="text-xs text-gray-400">{f.child_count} children</span>
              </div>
              <p className="text-xs text-gray-500 font-mono">Root: {f.root_token_hash}</p>
              <div className="flex items-center gap-1 mt-2">
                <span className="text-xs px-1.5 py-0.5 bg-gray-700 rounded">root</span>
                <span className="text-gray-600">→</span>
                <span className="text-xs px-1.5 py-0.5 bg-gray-700 rounded">access_token</span>
                {f.child_count > 1 && <><span className="text-gray-600">→</span><span className="text-xs px-1.5 py-0.5 bg-gray-700 rounded">refresh_token</span></>}
              </div>
            </div>
          ))}
        </div>
      </div>

      {data?.reuse_alerts && data.reuse_alerts.length > 0 && (
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-sm font-semibold mb-3 flex items-center gap-2"><AlertTriangle className="w-4 h-4 text-red-400" /> Reuse Detection Alerts</h2>
          <div className="space-y-2">
            {data.reuse_alerts.map((a: any) => (
              <div key={a.id} className="flex items-center gap-3 bg-red-950 border border-red-800 rounded-lg p-3">
                <AlertTriangle className="w-4 h-4 text-red-400" />
                <div className="flex-1"><p className="text-sm">{a.description}</p><p className="text-xs text-gray-400">Family: {a.family_id} - Detected: {a.detected_at}</p></div>
                <span className="text-xs px-2 py-0.5 rounded bg-red-900 text-red-300">Cascade revoke recommended</span>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
