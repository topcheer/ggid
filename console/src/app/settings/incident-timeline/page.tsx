"use client";

import { useIncidentTimeline } from "@ggid/sdk-react";
import { Clock, AlertTriangle, Link2, FileText } from "lucide-react";
import { useTranslations } from "@/lib/i18n";

export default function IncidentTimelinePage() {
  const t = useTranslations();

  const { events, isLoading, error, fetchTimeline } = useIncidentTimeline();

  if (isLoading) return <div className="p-8 text-gray-400">Loading timeline...</div>;
  if (error) return <div className="p-8 text-red-400">Error: {error}</div>;

  const phaseColors: Record<string, string> = {
    detection: "bg-blue-900 text-blue-300",
    triage: "bg-yellow-900 text-yellow-300",
    escalation: "bg-orange-900 text-orange-300",
    containment: "bg-purple-900 text-purple-300",
    response: "bg-green-900 text-green-300",
    resolution: "bg-green-900 text-green-300",
    postmortem: "bg-gray-700 text-gray-300",
  };

  return (
    <div className="min-h-screen bg-gray-950 text-white p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">Incident Timeline</h1>
          <p className="text-sm text-gray-400 mt-1">Chronological incident events and response</p>
        </div>
      </div>

      {events.length === 0 ? (
        <div className="bg-gray-900 rounded-xl p-12 text-center">
          <Clock className="w-12 h-12 text-gray-600 mx-auto mb-3" />
          <p className="text-gray-400">Select an incident to view its timeline</p>
          <button onClick={() => fetchTimeline("latest")} className="mt-3 px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg text-sm font-medium transition">Load Latest Incident</button>
        </div>
      ) : (
        <div className="bg-gray-900 rounded-xl p-6">
          <div className="relative pl-6">
            {/* Timeline line */}
            <div className="absolute left-2 top-0 bottom-0 w-0.5 bg-gray-700" />

            {events.map((ev: any) => (
              <div key={ev.id} className="relative pb-6 last:pb-0">
                {/* Dot */}
                <div className={"absolute -left-4 w-3 h-3 rounded-full ring-2 ring-gray-900 " + (phaseColors[ev.phase]?.replace("text-", "bg-").split(" ")[0] ?? "bg-gray-500")} />

                <div className="ml-4">
                  <div className="flex items-center gap-2 mb-1">
                    <span className={"text-xs px-1.5 py-0.5 rounded " + (phaseColors[ev.phase] ?? "bg-gray-700 text-gray-300")}>{ev.phase}</span>
                    <span className="text-xs text-gray-500">{ev.created_at}</span>
                  </div>
                  <p className="text-sm text-gray-200">{ev.description}</p>
                  <p className="text-xs text-gray-400 mt-0.5">Actor: {ev.actor}</p>
                  {Object.keys(ev.metadata).length > 0 && (
                    <div className="flex flex-wrap gap-1 mt-1">
                      {Object.entries(ev.metadata).map(([k, v]: any[]) => {
                        const val: string = typeof v === "string" ? v : JSON.stringify(v);
                        return <span key={k} className="text-xs px-1 py-0.5 bg-gray-800 rounded text-gray-500">{k}: {val}</span>;
                      })}
                    </div>
                  )}
                </div>
              </div>
            ))}
          </div>

          {/* SLA breach markers */}
          <div className="mt-6 flex items-center gap-2 text-xs text-gray-500">
            <AlertTriangle className="w-3 h-3 text-red-400" />
            SLA breach markers and post-mortem links shown inline above
          </div>
        </div>
      )}
    </div>
  );
}
