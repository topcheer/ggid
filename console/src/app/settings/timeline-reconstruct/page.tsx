"use client";

import { useState, useCallback } from "react";
import { Clock, Link2, AlertTriangle, Eye } from "lucide-react";
import { useTranslations } from "@/lib/i18n";
import { authHeader, isAuthenticated } from "@/lib/auth-helpers";

interface TimelineEvent {
  id: string;
  timestamp: string;
  event_type: string;
  description: string;
  source: string;
  ip?: string;
  correlated_with?: string;
}

interface ReconstructData {
  events: TimelineEvent[];
  correlation_chains: { chain_id: string; event_ids: string[]; pattern: string }[];
  gaps: { after_event: string; gap_minutes: number; severity: "low" | "medium" | "high" }[];
  anomaly_windows: { start: string; end: string; type: string }[];
}

export default function TimelineReconstructPage() {
  const t = useTranslations();
  const [userId, setUserId] = useState("");
  const [sessionId, setSessionId] = useState("");
  const [data, setData] = useState<ReconstructData | null>(null);
  const [loading, setLoading] = useState(false);

  const reconstruct = useCallback(async () => {
    if (!userId && !sessionId) return;
    setLoading(true);
    try {
      const params = new URLSearchParams();
      if (userId) params.set("user_id", userId);
      if (sessionId) params.set("session_id", sessionId);
      const res = await fetch(`/api/v1/audit/timeline-reconstruct?${params}`, { headers: { ...authHeader(), "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) setData(await res.json());
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, [userId, sessionId]);

  const gapColors: Record<string, string> = { low: "text-gray-500", medium: "text-yellow-600", high: "text-red-600" };
  const correlatedIds = new Set(data?.correlation_chains.flatMap((c) => c.event_ids) || []);
  const gapAfterMap = new Map(data?.gaps.map((g) => [g.after_event, g]) || []);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><Clock className="w-6 h-6 text-purple-500" /> Timeline Reconstruct</h1>
        <p className="text-sm text-gray-500 mt-1">Reconstruct and correlate user session events with gap and anomaly detection.</p>
      </div>

      <div className="rounded-lg border dark:border-gray-800 p-4 space-y-3">
        <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
          <div><label className="text-sm font-medium">User ID</label><input aria-label="usr-xxxx" type="text" value={userId} onChange={(e) => setUserId(e.target.value)} placeholder="usr-xxxx" className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm font-mono" /></div>
          <div><label className="text-sm font-medium">Session ID</label><input aria-label="sess-xxxx" type="text" value={sessionId} onChange={(e) => setSessionId(e.target.value)} placeholder="sess-xxxx" className="w-full mt-1 px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm font-mono" /></div>
        </div>
        <button aria-label="Eye" onClick={reconstruct} disabled={loading || (!userId && !sessionId)} className="px-4 py-2 rounded-lg bg-purple-600 text-white text-sm font-medium hover:bg-purple-700 disabled:opacity-50 flex items-center gap-2"><Eye className="w-4 h-4" /> {loading ? "Reconstructing..." : "Reconstruct"}</button>
      </div>

      {data && (
        <>
          {data.anomaly_windows.length > 0 && (
            <div className="space-y-2">{data.anomaly_windows.map((w, i) => (
              <div key={i} className="rounded-lg border border-orange-200 dark:border-orange-800 bg-orange-50 dark:bg-orange-900/20 p-3 flex items-center gap-2"><AlertTriangle className="w-4 h-4 text-orange-500" /><span className="text-sm flex-1"><strong>Anomaly ({w.type}):</strong> {w.start} to {w.end}</span></div>
            ))}</div>
          )}

          <div className="relative pl-8">
            <div className="absolute left-3 top-0 bottom-0 w-px bg-gray-200 dark:bg-gray-800" />
            <div className="space-y-2">
              {data.events.map((evt) => {
                const isCorrelated = correlatedIds.has(evt.id);
                const gap = gapAfterMap.get(evt.id);
                return (
                  <div key={evt.id}>
                    <div className={`relative rounded-lg border p-3 ml-2 ${isCorrelated ? "border-purple-300 dark:border-purple-800" : "dark:border-gray-800"}`}>
                      <div className={`absolute -left-5 w-4 h-4 rounded-full border-2 ${isCorrelated ? "bg-purple-500 border-purple-200" : "bg-blue-500 border-blue-200"}`} />
                      <div className="flex items-center justify-between">
                        <div className="flex items-center gap-2"><span className="text-xs font-mono text-gray-400">{evt.timestamp}</span>{isCorrelated && <Link2 className="w-3 h-3 text-purple-500" />}</div>
                        <span className="px-2 py-0.5 rounded text-xs bg-gray-100 dark:bg-gray-800">{evt.event_type}</span>
                      </div>
                      <p className="text-sm mt-1">{evt.description}</p>
                      <div className="flex items-center gap-3 mt-1 text-xs text-gray-400"><span>{evt.source}</span>{evt.ip && <span>IP: {evt.ip}</span>}</div>
                    </div>
                    {gap && (
                      <div className={`ml-6 my-1 text-xs flex items-center gap-1 ${gapColors[gap.severity]}`}><Clock className="w-3 h-3" /> Gap: {gap.gap_minutes}min ({gap.severity})</div>
                    )}
                  </div>
                );
              })}
              {data.events.length === 0 && <p className="text-sm text-gray-500 py-4 ml-2">No events found.</p>}
            </div>
          </div>

          {data.correlation_chains.length > 0 && (
            <div className="rounded-lg border dark:border-gray-800 p-4">
              <h3 className="text-sm font-semibold flex items-center gap-2 mb-3"><Link2 className="w-4 h-4 text-purple-500" /> Correlation Chains</h3>
              <div className="space-y-2">{data.correlation_chains.map((c) => (
                <div key={c.chain_id} className="flex items-center gap-2 text-sm"><span className="font-mono text-xs text-purple-600">{c.chain_id}</span><span className="flex-1">{c.pattern}</span><span className="text-xs text-gray-400">{c.event_ids.length} events</span></div>
              ))}</div>
            </div>
          )}
        </>
      )}
    </div>
  );
}
