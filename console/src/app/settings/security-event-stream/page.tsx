"use client";

import { useState } from "react";
import { useSecurityEventStream } from "@ggid/sdk-react";
import { Activity, Pause, Play, Download } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

export default function SecurityEventStreamPage() {
  const t = useTranslations();

  const { data, loading, error, refresh } = useSecurityEventStream();
  const [paused, setPaused] = useState(false);
  const [severityFilter, setSeverityFilter] = useState("all");
  const [typeFilter, setTypeFilter] = useState("all");

  if (loading) return <div className="p-8 text-gray-400">Loading event stream...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  const sevColors: Record<string, string> = {
    critical: "bg-red-900 text-red-300",
    high: "bg-orange-900 text-orange-300",
    medium: "bg-yellow-900 text-yellow-300",
    low: "bg-blue-900 text-blue-300",
  };

  const events = (data?.events ?? []).filter((e) =>
    (severityFilter === "all" || e.severity === severityFilter) &&
    (typeFilter === "all" || e.type === typeFilter)
  );
  const allTypes: string[] = Array.from(new Set((data?.events ?? []).map((e) => e.type)));

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold">Security Event Stream</h1>
          <p className="text-sm text-gray-400 mt-1">Real-time security event monitoring</p>
        </div>
        <div className="flex items-center gap-2">
          <button onClick={() => setPaused(!paused)} className="flex items-center gap-1 px-3 py-2 bg-gray-700 hover:bg-gray-600 rounded-lg text-sm font-medium transition">
            {paused ? <><Play className="w-4 h-4" /> Resume</> : <><Pause className="w-4 h-4" /> Pause</>}
          </button>
          <button className="flex items-center gap-1 px-3 py-2 bg-gray-700 hover:bg-gray-600 rounded-lg text-sm font-medium transition">
            <Download className="w-4 h-4" /> Snapshot
          </button>
        </div>
      </div>

      {/* Filters */}
      <div className="flex items-center gap-3 mb-4">
        <select value={severityFilter} onChange={(e) => setSeverityFilter(e.target.value)} className="px-3 py-1.5 bg-gray-800 rounded-lg text-sm">
          <option value="all">All Severities</option><option value="critical">Critical</option><option value="high">High</option><option value="medium">Medium</option><option value="low">Low</option>
        </select>
        <select value={typeFilter} onChange={(e) => setTypeFilter(e.target.value)} className="px-3 py-1.5 bg-gray-800 rounded-lg text-sm">
          <option value="all">All Types</option>
          {allTypes.map((t) => <option key={t} value={t}>{t}</option>)}
        </select>
        <span className="text-xs text-gray-500 ml-auto">{events.length} events</span>
      </div>

      {/* Event Feed */}
      <div className="bg-gray-900 rounded-xl p-4">
        <div className="space-y-1 max-h-[600px] overflow-y-auto">
          {events.map((ev) => (
            <details key={ev.id} className="bg-gray-800 rounded-lg p-2 group">
              <summary className="flex items-center gap-3 cursor-pointer list-none">
                <span className={"text-xs px-1.5 py-0.5 rounded flex-shrink-0 " + (sevColors[ev.severity] ?? "bg-gray-700")}>{ev.severity}</span>
                <span className="text-xs font-mono text-gray-400 flex-shrink-0">{ev.type}</span>
                <span className="text-sm text-gray-300 flex-1 truncate">{ev.message}</span>
                <span className="text-xs text-gray-500 flex-shrink-0">{ev.timestamp}</span>
                {ev.correlation_id && <span className="text-xs px-1 py-0.5 bg-purple-900 text-purple-300 rounded">corr</span>}
              </summary>
              <div className="mt-2 ml-8 space-y-1">
                <p className="text-xs text-gray-400">Source: {ev.source}</p>
                {ev.affected_entities.length > 0 && (
                  <p className="text-xs text-gray-400">Affected: {ev.affected_entities.join(", ")}</p>
                )}
                {ev.raw_data && (
                  <pre className="bg-gray-900 rounded p-2 text-xs text-gray-500 font-mono overflow-x-auto">{ev.raw_data}</pre>
                )}
              </div>
            </details>
          ))}
          {events.length === 0 && <p className="text-sm text-gray-500 p-4 text-center">No events</p>}
        </div>
      </div>
    </div>
  );
}
