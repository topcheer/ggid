"use client";

import { useDLP } from "@ggid/sdk-react";
import { useTranslations } from "@/lib/i18n";
import { ShieldBan, Play, AlertTriangle } from "lucide-react";

export default function DLPPage() {
  const t = useTranslations();
  const { data, loading, error, refresh, testPolicy } = useDLP();

  if (loading) return <div className="p-8 text-gray-400">Loading DLP...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  const actionColors: Record<string, string> = {
    block: "bg-red-900 text-red-300",
    mask: "bg-yellow-900 text-yellow-300",
    log: "bg-blue-900 text-blue-300",
  };

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">Data Loss Prevention</h1>
          <p className="text-sm text-gray-400 mt-1">Monitor and prevent sensitive data exfiltration</p>
        </div>
        <button onClick={refresh} className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition">Refresh</button>
      </div>

      {/* DLP Policies */}
      <div className="bg-gray-900 rounded-xl p-6 mb-6">
        <h2 className="text-sm font-semibold flex items-center gap-2 mb-4">
          <ShieldBan className="w-4 h-4 text-purple-400" />
          DLP Policies
        </h2>
        <div className="space-y-2">
          {(data?.dlp_policies ?? []).map((p) => (
            <div key={p.policy_name} className="bg-gray-800 rounded-lg p-3">
              <div className="flex items-center justify-between mb-2">
                <div>
                  <p className="text-sm font-medium">{p.policy_name}</p>
                  <p className="text-xs text-gray-400">Trigger: {p.trigger_pattern}</p>
                </div>
                <span className={"text-xs px-2 py-0.5 rounded " + (actionColors[p.action] ?? "bg-gray-700")}>{p.action}</span>
              </div>
              <div className="flex gap-1">
                {p.channels.map((ch) => (
                  <span key={ch} className="text-xs px-1.5 py-0.5 bg-gray-700 rounded text-gray-400">{ch}</span>
                ))}
              </div>
            </div>
          ))}
        </div>
      </div>

      {/* Violation Log */}
      <div className="bg-gray-900 rounded-xl p-6">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-sm font-semibold flex items-center gap-2">
            <AlertTriangle className="w-4 h-4 text-red-400" />
            Violation Log (24h)
          </h2>
          <button onClick={() => testPolicy("test-input")} className="flex items-center gap-1 px-3 py-1 bg-gray-700 hover:bg-gray-600 rounded text-xs font-medium transition">
            <Play className="w-3 h-3" /> Test Policy
          </button>
        </div>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-gray-800 text-gray-400">
                <th scope="col" className="text-left py-2 pr-3">User</th>
                <th scope="col" className="text-left py-2 pr-3">Resource</th>
                <th scope="col" className="text-left py-2 pr-3">Pattern</th>
                <th scope="col" className="text-left py-2 pr-3">Action</th>
                <th scope="col" className="text-left py-2 pr-3">Timestamp</th>
              </tr>
            </thead>
            <tbody>
              {(data?.violation_log ?? []).map((v: any, i: number) => (
                <tr key={i} className="border-b border-gray-800">
                  <td className="py-3 pr-3 text-xs">{v.user}</td>
                  <td className="py-3 pr-3 text-xs text-gray-400">{v.resource}</td>
                  <td className="py-3 pr-3 text-xs font-mono text-purple-400">{v.pattern_matched}</td>
                  <td className="py-3 pr-3">
                    <span className={"text-xs px-2 py-0.5 rounded " + (actionColors[v.action_taken] ?? "bg-gray-700")}>{v.action_taken}</span>
                  </td>
                  <td className="py-3 pr-3 text-xs text-gray-500">{v.timestamp}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
