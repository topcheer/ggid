"use client";

import React, { useState } from "react";
import { useApi } from "@/lib/api";
import { GitCommitVertical, Loader2, AlertCircle, X, Radio, Shield, Eye, Wrench, CheckCircle, FileSearch } from "lucide-react";

interface TimelineEvent {
  id: string; incident_id: string; phase: string;
  description: string; actor: string;
  metadata: Record<string, string>; created_at: string;
}

interface Incident {
  id: string; title: string; severity: string; status: string;
}

const phaseIcons: Record<string, React.ReactNode> = {
  detection: <Radio className="h-4 w-4 text-blue-500" />,
  triage: <FileSearch className="h-4 w-4 text-purple-500" />,
  escalation: <AlertCircle className="h-4 w-4 text-orange-500" />,
  containment: <Shield className="h-4 w-4 text-yellow-500" />,
  response: <Wrench className="h-4 w-4 text-indigo-500" />,
  resolution: <CheckCircle className="h-4 w-4 text-green-500" />,
  postmortem: <FileSearch className="h-4 w-4 text-gray-500" />,
};

export default function IncidentTimelinePage() {
  const { apiFetch } = useApi();
  const [incidents, setIncidents] = useState<Incident[]>([]);
  const [selected, setSelected] = useState<string | null>(null);
  const [events, setEvents] = useState<TimelineEvent[]>([]);
  const [loading, setLoading] = useState(true);
  const [loadingTimeline, setLoadingTimeline] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useState(() => { (async () => { try { setIncidents(await apiFetch<Incident[]>("/api/v1/audit/incidents").catch(() => [])); } catch { setError("Failed to load incidents"); } finally { setLoading(false); } })(); });

  const handleSelect = async (id: string) => {
    setSelected(id); setLoadingTimeline(true); setEvents([]);
    try { setEvents(await apiFetch<TimelineEvent[]>(`/api/v1/audit/incidents/${id}/timeline`)); }
    catch { setError("Failed to load timeline"); }
    finally { setLoadingTimeline(false); }
  };

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  return (
    <div className="space-y-6">
      <div><h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><GitCommitVertical className="h-6 w-6 text-blue-600" /> Incident Timeline</h1><p className="mt-1 text-sm text-gray-500 dark:text-gray-400">Track incident lifecycle: detection, escalation, response, and resolution.</p></div>
      {error && <div className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400"><AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} className="ml-auto"><X className="h-4 w-4" /></button></div>}
      <div className="grid grid-cols-4 gap-6">
        {/* Incident selector */}
        <div className={cardCls}><h3 className="mb-3 text-xs font-semibold uppercase text-gray-400">Incidents</h3>{loading ? <Loader2 className="h-5 w-5 animate-spin text-blue-600" /> : incidents.length === 0 ? <p className="text-sm text-gray-400">No incidents.</p> : <div className="space-y-1">{incidents.map((inc) => <button key={inc.id} onClick={() => handleSelect(inc.id)} className={`flex w-full items-center gap-2 rounded px-2 py-2 text-left text-sm ${selected === inc.id ? "bg-blue-50 dark:bg-blue-900/20" : "hover:bg-gray-50 dark:hover:bg-gray-800"}`}><span className={`h-2 w-2 rounded-full ${inc.severity === "critical" ? "bg-red-500" : inc.severity === "high" ? "bg-orange-500" : "bg-yellow-500"}`} /><span className="truncate text-gray-700 dark:text-gray-300">{inc.title}</span></button>)}</div>}</div>
        {/* Timeline */}
        <div className="col-span-3">{!selected ? <div className={cardCls}><div className="py-12 text-center"><GitCommitVertical className="mx-auto h-12 w-12 text-gray-300" /><p className="mt-4 text-sm text-gray-400">Select an incident to view its timeline.</p></div></div> : loadingTimeline ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-blue-600" /></div> : (
          <div className="relative"><div className="absolute left-5 top-0 bottom-0 w-px bg-gray-200 dark:bg-gray-700" /><div className="space-y-4">{events.map((e) => (
            <div key={e.id} className="relative flex gap-4 pl-2"><div className="z-10 flex h-8 w-8 items-center justify-center rounded-full border-2 border-white bg-white dark:border-gray-800 dark:bg-gray-800">{phaseIcons[e.phase] || <Radio className="h-4 w-4 text-gray-400" />}</div><div className="flex-1 rounded-lg border border-gray-200 bg-white p-3 shadow-sm dark:border-gray-700 dark:bg-gray-800"><div className="flex items-center justify-between"><span className="text-xs font-semibold uppercase text-gray-400">{e.phase}</span><span className="text-xs text-gray-400">{new Date(e.created_at).toLocaleString()}</span></div><p className="mt-1 text-sm text-gray-600 dark:text-gray-300">{e.description}</p><div className="mt-1 text-xs text-gray-400">by {e.actor.slice(0, 12)}</div></div></div>
          ))}</div></div>
        )}</div>
      </div>
    </div>
  );
}
