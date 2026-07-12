"use client";

import { useState, useEffect, useRef, useCallback } from "react";
import { Radio, Pause, Play, Download } from "lucide-react";

interface LiveEvent {
  id: string;
  timestamp: string;
  user: string;
  action: string;
  resource: string;
  ip_address: string;
  result: "success" | "denied" | "error";
  severity: "info" | "warning" | "error" | "critical";
}

const sevColors: Record<string, string> = {
  info: "border-l-gray-400", warning: "border-l-yellow-500", error: "border-l-red-500", critical: "border-l-red-600",
};
const resultColors: Record<string, string> = {
  success: "text-green-600", denied: "text-yellow-600", error: "text-red-600",
};

export default function AuditRealtimePage() {
  const [events, setEvents] = useState<LiveEvent[]>([]);
  const [paused, setPaused] = useState(false);
  const [filterSeverity, setFilterSeverity] = useState("");
  const [filterType, setFilterType] = useState("");
  const feedRef = useRef<HTMLDivElement>(null);

  const fetchData = useCallback(async () => {
    if (paused) return;
    try {
      const res = await fetch("/api/v1/audit/realtime?limit=50", { headers: { "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) { const d = await res.json(); setEvents((d.events || d || []).slice(0, 50)); }
    } catch { /* noop */ }
  }, [paused]);

  useEffect(() => { fetchData(); const interval = setInterval(fetchData, 5000); return () => clearInterval(interval); }, [fetchData]);

  useEffect(() => { if (!paused && feedRef.current) feedRef.current.scrollTop = 0; }, [events, paused]);

  const exportSnapshot = () => { const json = JSON.stringify(events, null, 2); const blob = new Blob([json], { type: "application/json" }); const url = URL.createObjectURL(blob); const a = document.createElement("a"); a.href = url; a.download = "audit-snapshot-" + Date.now() + ".json"; a.click(); };

  const filtered = events.filter((e) => { if (filterSeverity && e.severity !== filterSeverity) return false; if (filterType && !e.action.includes(filterType)) return false; return true; });

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div><h1 className="text-2xl font-bold flex items-center gap-2"><Radio className={"w-6 h-6 text-red-500 " + (!paused ? "animate-pulse" : "")} /> Audit Realtime</h1><p className="text-sm text-gray-500 mt-1">Live audit event stream with filtering and export.</p></div>
        <div className="flex items-center gap-2"><button onClick={() => setPaused(!paused)} className={"px-3 py-1.5 rounded-lg text-sm font-medium flex items-center gap-1 " + (paused ? "bg-green-600 text-white" : "border dark:border-gray-700")}>{paused ? <><Play className="w-3.5 h-3.5" /> Resume</> : <><Pause className="w-3.5 h-3.5" /> Pause</>}</button><button onClick={exportSnapshot} className="px-3 py-1.5 rounded-lg border dark:border-gray-700 text-sm flex items-center gap-1"><Download className="w-3.5 h-3.5" /> Snapshot</button></div>
      </div>

      <div className="flex items-center gap-2">
        <select value={filterSeverity} onChange={(e) => setFilterSeverity(e.target.value)} className="px-3 py-1.5 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm"><option value="">All Severities</option><option value="info">Info</option><option value="warning">Warning</option><option value="error">Error</option><option value="critical">Critical</option></select>
        <input type="text" placeholder="Filter by action..." value={filterType} onChange={(e) => setFilterType(e.target.value)} className="px-3 py-1.5 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm w-48" />
        <span className="text-sm text-gray-500">{filtered.length} events {paused && "(paused)"}</span>
      </div>

      <div ref={feedRef} className="space-y-2 max-h-[60vh] overflow-y-auto">
        {filtered.map((e) => (
          <div key={e.id} className={"rounded-lg border-l-4 dark:border-gray-800 bg-white dark:bg-gray-900 p-3 " + sevColors[e.severity]}>
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2"><span className={"text-xs font-medium " + resultColors[e.result]}>{e.result}</span><span className="text-sm font-medium">{e.action}</span><span className="text-xs text-gray-500">{e.user}</span></div>
              <span className="text-xs text-gray-400">{e.timestamp}</span>
            </div>
            <div className="flex items-center gap-3 mt-1 text-xs text-gray-500"><span className="font-mono">{e.resource}</span><span>IP: {e.ip_address}</span></div>
          </div>
        ))}
        {filtered.length === 0 && <p className="text-sm text-gray-500 text-center py-8">No events. {paused ? "Feed is paused." : "Waiting for events..."}</p>}
      </div>
    </div>
  );
}
