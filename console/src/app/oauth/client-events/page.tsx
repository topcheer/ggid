"use client";

import { useState, useEffect, useCallback } from "react";
import { Activity, MonitorSmartphone, User, Clock, Plus, Key, Pause, Play, Edit3 } from "lucide-react";
import { useTranslations } from "@/lib/i18n";
import { authHeader, isAuthenticated } from "@/lib/auth-helpers";

interface ClientEvent {
  id: string;
  event_type: "created" | "updated" | "rotated" | "suspended" | "reinstated" | "deleted";
  actor: string;
  timestamp: string;
  details: string;
  metadata: Record<string, string>;
}

interface ClientSummary {
  client_id: string;
  client_name: string;
}

const eventIcons: Record<string, { icon: typeof Plus; color: string }> = {
  created: { icon: Plus, color: "bg-green-50 dark:bg-green-900/20 text-green-500" },
  updated: { icon: Edit3, color: "bg-blue-50 dark:bg-blue-900/20 text-blue-500" },
  rotated: { icon: Key, color: "bg-purple-50 dark:bg-purple-900/20 text-purple-500" },
  suspended: { icon: Pause, color: "bg-red-50 dark:bg-red-900/20 text-red-500" },
  reinstated: { icon: Play, color: "bg-yellow-50 dark:bg-yellow-900/20 text-yellow-500" },
  deleted: { icon: Pause, color: "bg-gray-100 dark:bg-gray-800 text-gray-500" },
};

export default function ClientEventsPage() {
  const t = useTranslations();

  const [clients, setClients] = useState<ClientSummary[]>([]);
  const [selectedId, setSelectedId] = useState("");
  const [events, setEvents] = useState<ClientEvent[]>([]);
  const [loading, setLoading] = useState(false);

  const fetchClients = useCallback(async () => {
    try {
      const res = await fetch("/api/v1/oauth/clients", { headers: { ...authHeader(), "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) {
        const data = await res.json();
        setClients(data.clients || data || []);
      }
    } catch { /* noop */ }
  }, []);

  const fetchEvents = useCallback(async () => {
    if (!selectedId) return;
    setLoading(true);
    try {
      const res = await fetch(`/api/v1/oauth/clients/${selectedId}/events`, { headers: { ...authHeader(), "X-Tenant-ID": "00000000-0000-0000-0000-000000000001" } });
      if (res.ok) {
        const data = await res.json();
        setEvents(data.events || data || []);
      }
    } catch { /* noop */ }
    finally { setLoading(false); }
  }, [selectedId]);

  useEffect(() => { fetchClients(); }, [fetchClients]);
  useEffect(() => { if (selectedId) fetchEvents(); }, [selectedId, fetchEvents]);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold flex items-center gap-2"><Activity className="w-6 h-6 text-blue-500" /> {t("oauthClientEvents.title")}</h1>
        <p className="text-sm text-gray-500 mt-1">Track OAuth client lifecycle events with actor attribution.</p>
      </div>

      <select aria-label="Selected id" value={selectedId} onChange={(e) => setSelectedId(e.target.value)} className="px-3 py-2 rounded-lg border dark:border-gray-700 dark:bg-gray-900 text-sm">
        <option value="">Select a client...</option>
        {clients.map((c: any) => <option key={c.client_id} value={c.client_id}>{c.client_name} ({c.client_id})</option>)}
      </select>

      {events.length > 0 && !loading && (
        <div className="rounded-lg border dark:border-gray-800">
          <div className="px-4 py-3 border-b dark:border-gray-800">
            <h3 className="font-semibold flex items-center gap-2"><Activity className="w-4 h-4" /> Event Timeline ({events.length})</h3>
          </div>
          <div className="relative max-h-[600px] overflow-y-auto">
            {events.map((evt: any, i: any) => {
              const cfg = eventIcons[evt.event_type] || eventIcons.updated;
              const Icon = cfg.icon;
              return (
                <div key={evt.id} className="relative flex gap-3 px-4 py-3">
                  {i < events.length - 1 && <div className="absolute left-[27px] top-14 bottom-0 w-0.5 bg-gray-200 dark:bg-gray-800" />}
                  <div className={`relative z-10 w-10 h-10 rounded-full flex items-center justify-center flex-shrink-0 ${cfg.color}`}>
                    <Icon className="w-5 h-5" />
                  </div>
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2">
                      <span className="font-medium capitalize text-sm">{evt.event_type}</span>
                      <span className="px-2 py-0.5 rounded text-xs bg-gray-100 dark:bg-gray-800 font-mono">{evt.event_type}</span>
                    </div>
                    <p className="text-sm text-gray-600 dark:text-gray-400 mt-0.5">{evt.details}</p>
                    <div className="flex items-center gap-3 text-xs text-gray-400 mt-1">
                      <span className="flex items-center gap-1"><User className="w-3 h-3" /> {evt.actor}</span>
                      <span className="flex items-center gap-1"><Clock className="w-3 h-3" /> {evt.timestamp}</span>
                    </div>
                    {Object.keys(evt.metadata).length > 0 && (
                      <div className="flex flex-wrap gap-1 mt-1">
                        {Object.entries(evt.metadata).map(([k, v]) => <span key={k} className="px-1.5 py-0.5 rounded text-xs bg-gray-100 dark:bg-gray-800 font-mono">{k}: {v}</span>)}
                      </div>
                    )}
                  </div>
                </div>
              );
            })}
          </div>
        </div>
      )}

      {events.length === 0 && !loading && selectedId && <p className="text-sm text-gray-500 text-center py-8">No events found for this client.</p>}
      {!selectedId && <p className="text-sm text-gray-500 text-center py-8">Select a client to view its event history.</p>}
      {loading && <p className="text-sm text-gray-500">Loading...</p>}
    </div>
  );
}
