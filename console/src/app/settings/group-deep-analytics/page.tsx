"use client";

import { useState } from "react";
import { useGroupDeepAnalytics } from "@ggid/sdk-react";
import { Layers, Grid, AlertTriangle } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

export default function GroupDeepAnalyticsPage() {
  const t = useTranslations();

  const { data, loading, error, refresh } = useGroupDeepAnalytics();
  const [selectedGroup, setSelectedGroup] = useState("");

  if (loading) return <div className="p-8 text-gray-400">Loading group analytics...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  const group = data?.groups?.find((g: any) => g.name === selectedGroup) ?? data?.groups?.[0];

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">Group Deep Analytics</h1>
          <p className="text-sm text-gray-400 mt-1">Activity, permissions, and structure analysis</p>
        </div>
        <button onClick={refresh} className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition">Refresh</button>
      </div>

      {/* Group Selector */}
      <div className="mb-6">
        <select aria-label="Selected group" value={selectedGroup} onChange={(e) => setSelectedGroup(e.target.value)} className="px-3 py-2 bg-gray-800 rounded-lg text-sm">
          {(data?.groups ?? []).map((g: any) => <option key={g.name} value={g.name}>{g.name} ({g.member_count} members)</option>)}
        </select>
      </div>

      {group && (
        <>
          {/* Summary */}
          <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
            <div className="bg-gray-900 rounded-xl p-4"><Layers className="w-5 h-5 text-blue-400 mb-1" /><p className="text-xs text-gray-400">Nesting Depth</p><p className="text-xl font-bold">{group.nesting_depth}</p></div>
            <div className="bg-gray-900 rounded-xl p-4"><p className="text-xs text-gray-400">Members</p><p className="text-xl font-bold">{group.member_count}</p></div>
            <div className="bg-gray-900 rounded-xl p-4"><p className="text-xs text-gray-400">Permissions</p><p className="text-xl font-bold">{group.permission_count}</p></div>
            <div className="bg-gray-900 rounded-xl p-4"><p className="text-xs text-gray-400">Activity Score</p><p className={"text-xl font-bold " + (group.activity_score >= 70 ? "text-green-400" : "text-yellow-400")}>{group.activity_score}%</p></div>
          </div>

          {/* Activity Heatmap */}
          <div className="bg-gray-900 rounded-xl p-6 mb-6">
            <h2 className="text-sm font-semibold mb-3 flex items-center gap-2"><Grid className="w-4 h-4 text-green-400" /> Member Activity Heatmap (7d x 24h)</h2>
            <div className="overflow-x-auto">
              <div className="inline-grid gap-px" style={{ gridTemplateColumns: "auto repeat(24, 1fr)" }}>
                <div />{Array.from({ length: 24 }, (_, h) => <div key={h} className="text-xs text-gray-500 text-center w-6">{h}</div>)}
                {group.heatmap.map((row, dayIdx) => (
                  <>{<div key={"d"+dayIdx} className="text-xs text-gray-500 pr-2">{["Mon","Tue","Wed","Thu","Fri","Sat","Sun"][dayIdx]}</div>}{row.map((val: any, h: number) => {
                    const intensity = Math.min(val / 10, 1);
                    const color = intensity > 0.7 ? "bg-green-600" : intensity > 0.3 ? "bg-green-800" : intensity > 0 ? "bg-green-950" : "bg-gray-800";
                    return <div key={dayIdx+"-"+h} className={"w-6 h-5 rounded-sm " + color} title={val + " events"} />;
                  })}</>
                ))}
              </div>
            </div>
          </div>

          {/* Role Distribution + Anomalies */}
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            <div className="bg-gray-900 rounded-xl p-6">
              <h2 className="text-sm font-semibold mb-3">Role Distribution</h2>
              <div className="space-y-2">
                {group.role_distribution.map((r: any) => (
                  <div key={r.role} className="flex items-center gap-2">
                    <span className="text-xs w-24">{r.role}</span>
                    <div className="flex-1 bg-gray-800 rounded-full h-3"><div className="bg-blue-600 h-3 rounded-full" style={{ width: r.pct + "%" }} /></div>
                    <span className="text-xs text-gray-400">{r.pct}%</span>
                  </div>
                ))}
              </div>
            </div>
            <div className="bg-gray-900 rounded-xl p-6">
              <h2 className="text-sm font-semibold mb-3 flex items-center gap-2"><AlertTriangle className="w-4 h-4 text-yellow-400" /> Access Pattern Anomalies</h2>
              <div className="space-y-2">
                {group.anomalies.map((a: any, i: number) => (
                  <div key={i} className="bg-gray-800 rounded p-2 text-xs"><span className="text-yellow-400">{a}</span></div>
                ))}
              </div>
            </div>
          </div>
        </>
      )}
    </div>
  );
}
