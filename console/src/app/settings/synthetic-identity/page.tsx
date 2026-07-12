"use client";

import { useSyntheticIdentity } from "@ggid/sdk-react";
import { UserX, Mail, Ban, Shield, AlertTriangle } from "lucide-react";

export default function SyntheticIdentityPage() {
  const { data, loading, error, refresh } = useSyntheticIdentity();

  if (loading) return <div className="p-8 text-gray-400">Loading synthetic identity...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">Synthetic Identity Detection</h1>
          <p className="text-sm text-gray-400 mt-1">Detect fraudulent accounts created with synthetic identities</p>
        </div>
        <button onClick={refresh} className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition">Refresh</button>
      </div>

      {/* Summary */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
        <div className="bg-gray-900 rounded-xl p-4">
          <UserX className="w-5 h-5 text-red-400 mb-1" />
          <p className="text-xs text-gray-400">Flagged Accounts</p>
          <p className="text-xl font-bold text-red-400">{data?.flagged_accounts?.length ?? 0}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <Mail className="w-5 h-5 text-yellow-400 mb-1" />
          <p className="text-xs text-gray-400">Disposable Domains</p>
          <p className="text-xl font-bold">{data?.disposable_domains_blocklist?.length ?? 0}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <Ban className="w-5 h-5 text-orange-400 mb-1" />
          <p className="text-xs text-gray-400">Auto-Block</p>
          <p className="text-sm font-bold">{data?.auto_block_enabled ? "Enabled" : "Disabled"}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <Shield className="w-5 h-5 text-blue-400 mb-1" />
          <p className="text-xs text-gray-400">Detection Rules</p>
          <p className="text-xl font-bold">{data?.detection_rules?.filter((r) => r.enabled).length ?? 0}</p>
        </div>
      </div>

      {/* Flagged Accounts */}
      <div className="bg-gray-900 rounded-xl p-6 mb-6">
        <h2 className="text-lg font-semibold mb-4">Flagged Accounts</h2>
        <div className="space-y-2">
          {(data?.flagged_accounts ?? []).map((a, i) => (
            <div key={i} className="flex items-center gap-4 bg-gray-800 rounded-lg p-3">
              <AlertTriangle className="w-4 h-4 text-red-400 flex-shrink-0" />
              <div className="flex-1 min-w-0">
                <p className="text-sm font-medium truncate">{a.email}</p>
                <p className="text-xs text-gray-400">Source: {a.registration_source} - Age: {a.account_age_hours}h</p>
              </div>
              {a.disposable_domain && (
                <span className="text-xs px-2 py-0.5 bg-red-900 text-red-300 rounded">Disposable</span>
              )}
              <div className="flex items-center gap-2">
                <div className="w-10 h-1.5 bg-gray-700 rounded-full">
                  <div className={"h-full rounded-full " + (a.risk_score > 70 ? "bg-red-500" : "bg-yellow-500")} style={{ width: a.risk_score + "%" }} />
                </div>
                <span className={"text-sm font-bold " + (a.risk_score > 70 ? "text-red-400" : "text-yellow-400")}>{a.risk_score}</span>
              </div>
              <button className="text-xs text-red-400 hover:underline">Block</button>
            </div>
          ))}
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Detection Rules */}
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-sm font-semibold mb-4">Detection Rules</h2>
          <div className="space-y-2">
            {(data?.detection_rules ?? []).map((r) => (
              <div key={r.rule_name} className="flex items-center justify-between bg-gray-800 rounded-lg p-3">
                <div>
                  <p className="text-sm font-medium">{r.rule_name}</p>
                  <p className="text-xs text-gray-400">{r.description}</p>
                </div>
                <span className={"text-xs px-2 py-0.5 rounded " + (r.enabled ? "bg-green-900 text-green-300" : "bg-gray-700 text-gray-400")}>
                  {r.enabled ? "On" : "Off"}
                </span>
              </div>
            ))}
          </div>
        </div>

        {/* Disposable Domains Blocklist */}
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-sm font-semibold flex items-center gap-2 mb-4">
            <Ban className="w-4 h-4 text-red-400" />
            Disposable Domains Blocklist
          </h2>
          <div className="flex flex-wrap gap-2">
            {(data?.disposable_domains_blocklist ?? []).map((d) => (
              <span key={d} className="text-xs px-2 py-1 bg-gray-800 rounded font-mono text-gray-400">{d}</span>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}
