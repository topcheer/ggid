"use client";

import { useLockoutPolicyConfig } from "@ggid/sdk-react";
import { Lock, AlertTriangle } from "lucide-react";

export default function LockoutPolicyConfigPage() {
  const { data, loading, error, refresh } = useLockoutPolicyConfig();
  if (loading) return <div className="p-8 text-gray-400">Loading lockout policy...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div><h1 className="text-2xl font-bold">Lockout Policy</h1><p className="text-sm text-gray-400 mt-1">Brute-force protection configuration</p></div>
        <button onClick={refresh} className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition">Save</button>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-6">
        <div className="bg-gray-900 rounded-xl p-4"><Lock className="w-5 h-5 text-red-400 mb-1" /><p className="text-xs text-gray-400">Max Failed Attempts</p><p className="text-2xl font-bold">{data?.max_failed_attempts}</p></div>
        <div className="bg-gray-900 rounded-xl p-4"><p className="text-xs text-gray-400">Lockout Duration</p><p className="text-2xl font-bold">{data?.lockout_duration_minutes}m</p></div>
        <div className="bg-gray-900 rounded-xl p-4"><AlertTriangle className="w-5 h-5 text-yellow-400 mb-1" /><p className="text-xs text-gray-400">Current Lockouts</p><p className="text-2xl font-bold text-red-400">{data?.current_lockouts?.length ?? 0}</p></div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        <div className="bg-gray-900 rounded-xl p-6 space-y-3">
          <h2 className="text-sm font-semibold">Settings</h2>
          <div><label className="text-xs text-gray-400">Max Failed Attempts: {data?.max_failed_attempts}</label><input type="range" min="3" max="20" defaultValue={data?.max_failed_attempts} className="w-full mt-1" /></div>
          <div><label className="text-xs text-gray-400">Lockout Duration (min)</label><input type="number" defaultValue={data?.lockout_duration_minutes} className="w-full mt-1 px-3 py-2 bg-gray-800 rounded-lg text-sm" /></div>
          <div className="flex items-center gap-2"><input type="checkbox" defaultChecked={data?.progressive_backoff} id="pb" /><label htmlFor="pb" className="text-sm">Progressive backoff</label></div>
          <div><label className="text-xs text-gray-400">Captcha After</label><input type="number" defaultValue={data?.captcha_after} className="w-full mt-1 px-3 py-2 bg-gray-800 rounded-lg text-sm" /></div>
          <div><label className="text-xs text-gray-400">Auto-Unlock After (min)</label><input type="number" defaultValue={data?.auto_unlock_after} className="w-full mt-1 px-3 py-2 bg-gray-800 rounded-lg text-sm" /></div>
        </div>
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-sm font-semibold mb-3">Current Lockouts</h2>
          <div className="space-y-1">{(data?.current_lockouts ?? []).map((l) => (
            <div key={l.username} className="flex items-center gap-2 bg-gray-800 rounded p-2 text-xs"><span className="flex-1">{l.username}</span><span className="text-gray-400">{l.attempts} attempts</span><span className="text-red-400">unlocks {l.unlock_at}</span></div>
          ))}{data?.current_lockouts?.length === 0 && <p className="text-xs text-gray-500">No active lockouts</p>}</div>
        </div>
      </div>

      <div className="bg-gray-900 rounded-xl p-6 mt-6">
        <h2 className="text-sm font-semibold mb-3">Per-Endpoint Configuration</h2>
        <table className="w-full text-sm"><thead><tr className="border-b border-gray-800 text-gray-400"><th className="text-left py-2">Endpoint</th><th className="text-left py-2">Threshold</th><th className="text-left py-2">Lockout</th></tr></thead>
          <tbody>{(data?.per_endpoint ?? []).map((e) => (<tr key={e.endpoint} className="border-b border-gray-800"><td className="py-2 font-mono text-xs">{e.endpoint}</td><td className="py-2">{e.threshold}</td><td className="py-2">{e.lockout_min}m</td></tr>))}</tbody>
        </table>
      </div>
    </div>
  );
}
