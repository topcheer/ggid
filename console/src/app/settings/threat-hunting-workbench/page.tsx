"use client";

import { useThreatHunting } from "@ggid/sdk-react";
import { Search, Crosshair, Bookmark, FileText, Users } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

export default function ThreatHuntingWorkbenchPage() {
  const t = useTranslations();

  const { data, loading, error, refresh } = useThreatHunting();

  if (loading) return <div className="p-8 text-gray-400">Loading threat hunting...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">Threat Hunting Workbench</h1>
          <p className="text-sm text-gray-400 mt-1">Proactive threat hunting with IOC queries and hypotheses</p>
        </div>
        <button onClick={refresh} aria-label="Refresh threat hunting data" className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition">Refresh</button>
      </div>

      {/* Query Builder */}
      <div className="bg-gray-900 rounded-xl p-6 mb-6">
        <h2 className="text-sm font-semibold flex items-center gap-2 mb-4">
          <Search className="w-4 h-4 text-blue-400" />
          Hunt Query Builder
        </h2>
        <div className="space-y-3">
          <div className="flex gap-2">
            <select aria-label="Select option" className="px-3 py-2 bg-gray-800 rounded-lg text-sm">
              <option>IOC Type</option><option>IP Address</option><option>User</option><option>Hash</option><option>Domain</option>
            </select>
            <select aria-label="Select option" className="px-3 py-2 bg-gray-800 rounded-lg text-sm">
              <option>Operator</option><option>equals</option><option>contains</option><option>matches regex</option>
            </select>
            <input aria-label="Enter value..." className="flex-1 px-3 py-2 bg-gray-800 rounded-lg text-sm" placeholder="Enter value..." />
            <select aria-label="Select option" className="px-3 py-2 bg-gray-800 rounded-lg text-sm">
              <option>Last 24h</option><option>Last 7d</option><option>Last 30d</option>
            </select>
            <button className="px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium">Run Hunt</button>
          </div>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Hunt Results */}
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-sm font-semibold flex items-center gap-2 mb-4">
            <Crosshair className="w-4 h-4 text-red-400" />
            Hunt Results
          </h2>
          <div className="space-y-2 max-h-72 overflow-y-auto">
            {(data?.hunt_results ?? []).map((r: any, i: number) => (
              <div key={i} className="bg-gray-800 rounded-lg p-3">
                <div className="flex items-start justify-between">
                  <p className="text-sm font-mono text-blue-400">{r.entity}</p>
                  <span className={"text-xs px-2 py-0.5 rounded " + (r.severity === "high" ? "bg-red-900 text-red-300" : "bg-yellow-900 text-yellow-300")}>{r.severity}</span>
                </div>
                <p className="text-xs text-gray-400 mt-1">{r.description}</p>
                <p className="text-xs text-gray-500 mt-0.5">{r.timestamp}</p>
              </div>
            ))}
            {(data?.hunt_results?.length ?? 0) === 0 && <p className="text-sm text-gray-500">No results</p>}
          </div>
        </div>

        {/* Hypothesis Tracker */}
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-sm font-semibold flex items-center gap-2 mb-4">
            <FileText className="w-4 h-4 text-yellow-400" />
            Hypothesis Tracker
          </h2>
          <div className="space-y-2">
            {(data?.hypotheses ?? []).map((h: any) => (
              <div key={h.id} className="bg-gray-800 rounded-lg p-3">
                <div className="flex items-start justify-between mb-1">
                  <p className="text-sm font-medium">{h.hypothesis}</p>
                  <span className={"text-xs px-2 py-0.5 rounded " + (
                    h.status === "confirmed" ? "bg-red-900 text-red-300" :
                    h.status === "disproven" ? "bg-gray-700 text-gray-400" :
                    "bg-yellow-900 text-yellow-300"
                  )}>{h.status}</span>
                </div>
                <p className="text-xs text-gray-400">Evidence: {h.evidence_count} items</p>
                {h.conclusion && <p className="text-xs text-gray-500 mt-1">{h.conclusion}</p>}
              </div>
            ))}
          </div>
        </div>
      </div>

      {/* Saved Hunts + Watchlist */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 mt-6">
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-sm font-semibold flex items-center gap-2 mb-3">
            <Bookmark className="w-4 h-4 text-green-400" />
            Saved Hunts
          </h2>
          <div className="space-y-1">
            {(data?.saved_hunts ?? []).map((h: any, i: number) => (
              <div key={i} className="flex items-center justify-between bg-gray-800 rounded-lg p-2">
                <span className="text-sm">{h.name}</span>
                <span className="text-xs text-gray-500">{h.last_run}</span>
              </div>
            ))}
          </div>
        </div>
        <div className="bg-gray-900 rounded-xl p-6">
          <h2 className="text-sm font-semibold flex items-center gap-2 mb-3">
            <Users className="w-4 h-4 text-purple-400" />
            Watchlist
          </h2>
          <div className="flex flex-wrap gap-2">
            {(data?.watchlist ?? []).map((w: any, i: number) => (
              <span key={i} className="text-xs px-2 py-1 bg-gray-800 rounded font-mono text-gray-400">{w}</span>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}
