"use client";

import { useState } from "react";
import { useVelocityRulesConfig } from "@ggid/sdk-react";
import { Gauge, Plus, Activity, Globe } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

export default function VelocityRulesConfigPage() {
  const t = useTranslations();

  const { data, loading, error, refresh } = useVelocityRulesConfig();
  const [showModal, setShowModal] = useState(false);

  if (loading) return <div className="p-8 text-gray-400">Loading velocity rules...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">Velocity Rules</h1>
          <p className="text-sm text-gray-400 mt-1">Rate-based abuse prevention rules</p>
        </div>
        <div className="flex items-center gap-2">
          <button onClick={() => setShowModal(true)} className="flex items-center gap-2 px-4 py-2 bg-green-600 hover:bg-green-700 rounded-lg text-sm font-medium transition">
            <Plus className="w-4 h-4" /> Add Rule
          </button>
          <button onClick={refresh} className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition">Refresh</button>
        </div>
      </div>

      {/* Summary */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
        <div className="bg-gray-900 rounded-xl p-4">
          <Gauge className="w-5 h-5 text-blue-400 mb-1" />
          <p className="text-xs text-gray-400">Active Rules</p>
          <p className="text-xl font-bold">{data?.rules?.length ?? 0}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <Activity className="w-5 h-5 text-red-400 mb-1" />
          <p className="text-xs text-gray-400">Triggered (24h)</p>
          <p className="text-xl font-bold text-red-400">{data?.rules?.reduce((a, r) => a + r.triggered_24h, 0) ?? 0}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <Globe className="w-5 h-5 text-purple-400 mb-1" />
          <p className="text-xs text-gray-400">Geo Velocity</p>
          <p className="text-sm font-bold">{data?.geographic_velocity_check ? "Enabled" : "Disabled"}</p>
        </div>
        <div className="bg-gray-900 rounded-xl p-4">
          <Activity className="w-5 h-5 text-green-400 mb-1" />
          <p className="text-xs text-gray-400">Scope</p>
          <p className="text-sm font-bold capitalize">{data?.scope ?? "per_ip"}</p>
        </div>
      </div>

      {/* Rules Table */}
      <div className="bg-gray-900 rounded-xl p-6">
        <h2 className="text-lg font-semibold mb-4">Velocity Rules</h2>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-gray-800 text-gray-400">
                <th scope="col" className="text-left py-2 pr-3">Rule Name</th>
                <th scope="col" className="text-left py-2 pr-3">Metric</th>
                <th scope="col" className="text-left py-2 pr-3">Window</th>
                <th scope="col" className="text-left py-2 pr-3">Threshold</th>
                <th scope="col" className="text-left py-2 pr-3">Current Rate</th>
                <th scope="col" className="text-left py-2 pr-3">Action</th>
                <th scope="col" className="text-right py-2 pr-3">Triggered (24h)</th>
              </tr>
            </thead>
            <tbody>
              {(data?.rules ?? []).map((r) => {
                const pct = r.threshold > 0 ? (r.current_rate / r.threshold) * 100 : 0;
                return (
                  <tr key={r.rule_name} className="border-b border-gray-800">
                    <td className="py-3 pr-3 text-sm font-medium">{r.rule_name}</td>
                    <td className="py-3 pr-3 text-xs text-gray-400">{r.metric}</td>
                    <td className="py-3 pr-3 text-xs">{r.window}</td>
                    <td className="py-3 pr-3 text-xs font-medium">{r.threshold}</td>
                    <td className="py-3 pr-3">
                      <div className="flex items-center gap-2 w-32">
                        <div className="flex-1 h-1.5 bg-gray-700 rounded-full">
                          <div className={"h-full rounded-full " + (pct > 80 ? "bg-red-500" : pct > 50 ? "bg-yellow-500" : "bg-green-500")} style={{ width: Math.min(pct, 100) + "%" }} />
                        </div>
                        <span className="text-xs">{r.current_rate}</span>
                      </div>
                    </td>
                    <td className="py-3 pr-3">
                      <span className={"text-xs px-2 py-0.5 rounded " + (
                        r.action === "block" ? "bg-red-900 text-red-300" :
                        r.action === "throttle" ? "bg-yellow-900 text-yellow-300" :
                        "bg-blue-900 text-blue-300"
                      )}>
                        {r.action}
                      </span>
                    </td>
                    <td className="py-3 pr-3 text-right">
                      <span className={"text-xs " + (r.triggered_24h > 0 ? "text-red-400 font-medium" : "text-gray-400")}>
                        {r.triggered_24h}
                      </span>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      </div>

      {/* Add Rule Modal */}
      {showModal && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50" onClick={() => setShowModal(false)}>
          <div className="bg-gray-900 rounded-xl p-6 max-w-md w-full mx-4" onClick={(e) => e.stopPropagation()}>
            <h3 className="text-lg font-semibold mb-4">Add Velocity Rule</h3>
            <div className="space-y-3">
              <div>
                <label className="text-xs text-gray-400">Rule Name</label>
                <input className="w-full mt-1 px-3 py-2 bg-gray-800 rounded-lg text-sm" placeholder="e.g. Rapid signups" />
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="text-xs text-gray-400">Metric</label>
                  <select className="w-full mt-1 px-3 py-2 bg-gray-800 rounded-lg text-sm">
                    <option>registrations</option><option>logins</option><option>password_changes</option><option>api_calls</option>
                  </select>
                </div>
                <div>
                  <label className="text-xs text-gray-400">Window</label>
                  <select className="w-full mt-1 px-3 py-2 bg-gray-800 rounded-lg text-sm">
                    <option>per_minute</option><option>per_hour</option><option>per_day</option>
                  </select>
                </div>
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="text-xs text-gray-400">Threshold</label>
                  <input type="number" defaultValue={10} className="w-full mt-1 px-3 py-2 bg-gray-800 rounded-lg text-sm" />
                </div>
                <div>
                  <label className="text-xs text-gray-400">Action</label>
                  <select className="w-full mt-1 px-3 py-2 bg-gray-800 rounded-lg text-sm">
                    <option>throttle</option><option>block</option><option>challenge</option>
                  </select>
                </div>
              </div>
              <div>
                <label className="text-xs text-gray-400">Scope</label>
                <div className="flex gap-2 mt-1">
                  {"per_ip".split("|").map((s) => (
                    <span key={s} className="text-xs px-3 py-1.5 bg-gray-800 rounded-lg border border-gray-700">{s}</span>
                  ))}
                  <span className="text-xs px-3 py-1.5 bg-gray-800 rounded-lg border border-gray-700">per_ip</span>
                  <span className="text-xs px-3 py-1.5 bg-gray-800 rounded-lg border border-gray-700">per_user</span>
                  <span className="text-xs px-3 py-1.5 bg-blue-900 rounded-lg border border-blue-700 text-blue-300">per_device</span>
                </div>
              </div>
            </div>
            <div className="flex gap-2 mt-4">
              <button onClick={() => setShowModal(false)} className="flex-1 px-4 py-2 bg-green-600 hover:bg-green-700 rounded-lg text-sm font-medium">Create Rule</button>
              <button onClick={() => setShowModal(false)} className="px-4 py-2 bg-gray-700 hover:bg-gray-600 rounded-lg text-sm">Cancel</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
