"use client";

import { useUserLifecycleStats } from "@ggid/sdk-react";
import { Users } from "lucide-react";

export default function UserLifecycleStatsPage() {
  const { data, loading, error, refresh } = useUserLifecycleStats();

  if (loading) return <div className="p-8 text-gray-400">Loading lifecycle stats...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  const stages = data?.stages ?? { active: 0, dormant: 0, suspended: 0, deactivated: 0, pending: 0 };
  const stageEntries: [string, number][] = Object.entries(stages) as [string, number][];
  const total = stageEntries.reduce((a, [, c]) => a + c, 0);
  const colors: Record<string, string> = { active: "#3b82f6", dormant: "#a78bfa", suspended: "#f59e0b", deactivated: "#ef4444", pending: "#6b7280" };

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div><h1 className="text-2xl font-bold">User Lifecycle Statistics</h1><p className="text-sm text-gray-400 mt-1">Track users across lifecycle stages</p></div>
        <button onClick={refresh} className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition">Refresh</button>
      </div>

      {/* Donut + Legend */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-6 mb-6">
        <div className="bg-gray-900 rounded-xl p-6 flex items-center justify-center">
          <svg width="200" height="200" viewBox="0 0 200 200">
            {(() => { let offset = 0; const r = 70; const circ = 2 * Math.PI * r;
              return stageEntries.map(([stage, count]) => {
                const pct = total > 0 ? count / total : 0;
                const dash = pct * circ;
                const elem = <circle key={stage} cx="100" cy="100" r={r} fill="none" stroke={colors[stage] ?? "#374151"} strokeWidth="20" strokeDasharray={dash + " " + (circ - dash)} strokeDashoffset={-offset} transform="rotate(-90 100 100)" />;
                offset += dash;
                return elem;
              });
            })()}
            <text x="100" y="95" textAnchor="middle" className="fill-white text-2xl font-bold">{total.toLocaleString()}</text>
            <text x="100" y="115" textAnchor="middle" className="fill-gray-400 text-xs">Total Users</text>
          </svg>
        </div>
        <div className="bg-gray-900 rounded-xl p-6">
          <h3 className="text-sm font-semibold mb-3">By Stage</h3>
          {stageEntries.map(([stage, count]) => (
            <div key={stage} className="flex items-center gap-2 mb-2">
              <span className="w-3 h-3 rounded-full" style={{ backgroundColor: colors[stage] }} />
              <span className="text-sm capitalize">{stage}</span>
              <span className="text-sm text-gray-400 ml-auto">{count.toLocaleString()}</span>
            </div>
          ))}
        </div>
      </div>

      {/* Avg Time Per Stage */}
      <div className="bg-gray-900 rounded-xl p-6 mb-6">
        <h2 className="text-sm font-semibold mb-3">Avg Time Per Stage</h2>
        <div className="space-y-2">
          {(data?.avg_time_per_stage ?? []).map((s) => (
            <div key={s.stage} className="flex items-center gap-2">
              <span className="text-xs w-24 capitalize">{s.stage}</span>
              <div className="flex-1 bg-gray-800 rounded-full h-3"><div className="bg-blue-600 h-3 rounded-full" style={{ width: Math.min(s.avg_days / 30 * 100, 100) + "%" }} /></div>
              <span className="text-xs text-gray-400">{s.avg_days}d</span>
            </div>
          ))}
        </div>
      </div>

      {/* Transition Rules + Monthly Trend */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-sm font-semibold mb-3">Transition Rules</h2>
          <div className="space-y-1">
            {(data?.transition_rules ?? []).map((r) => (
              <div key={r.rule} className="flex items-center gap-2 bg-gray-800 rounded p-2 text-xs"><span className="text-gray-300">{r.rule}</span><span className="ml-auto text-gray-500">{r.trigger}</span></div>
            ))}
          </div>
        </div>
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-sm font-semibold mb-3">Monthly Transitions</h2>
          <div className="flex items-end gap-1 h-24">
            {(data?.monthly_transitions ?? []).map((m) => (
              <div key={m.month} className="flex-1 bg-blue-600 rounded-t" style={{ height: Math.abs(m.count) / 10 + "%" }} title={m.month + ": " + m.count} />
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}
