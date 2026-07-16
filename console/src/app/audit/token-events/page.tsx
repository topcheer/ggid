"use client";

import { useState, useRef, useEffect } from "react";
import { useApi } from "@/lib/api";
import {
  KeyRound, Loader2, AlertCircle, X, Play, Pause, Radio, Trash2,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface TokenEvent {
  id: string;
  event_type: "issued" | "refreshed" | "revoked" | "expired" | "introspected" | "exchanged";
  token_type: "access" | "refresh" | "id" | "agent";
  client_id: string;
  user_id: string;
  scopes: string[];
  ip_address: string;
  created_at: string;
}

const eventColors: Record<string, string> = {
  issued: "text-green-600 bg-green-100 dark:bg-green-900/30 dark:text-green-400",
  refreshed: "text-blue-600 bg-blue-100 dark:bg-blue-900/30 dark:text-blue-400",
  revoked: "text-red-600 bg-red-100 dark:bg-red-900/30 dark:text-red-400",
  expired: "text-gray-600 bg-gray-100 dark:bg-gray-700 dark:text-gray-400",
  introspected: "text-purple-600 bg-purple-100 dark:bg-purple-900/30 dark:text-purple-400",
  exchanged: "text-orange-600 bg-orange-100 dark:bg-orange-900/30 dark:text-orange-400",
};

export default function TokenEventsPage() {
  const t = useTranslations();

  const { apiFetch } = useApi();
  const [events, setEvents] = useState<TokenEvent[]>([]);
  const [connected, setConnected] = useState(false);
  const [paused, setPaused] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [filter, setFilter] = useState<string>("all");
  const esRef = useRef<EventSource | null>(null);
  const pausedRef = useRef(false);
  const reconnectRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const connect = () => {
    const tok = typeof window !== "undefined" ? localStorage.getItem("ggid_access_token") || "" : "";
    const baseUrl = typeof window !== "undefined" ? window.location.origin : "";
    const tenantId = typeof window !== "undefined" ? localStorage.getItem("ggid_tenant_id") || "00000000-0000-0000-0000-000000000001" : "";
    if (!tok || typeof window === "undefined" || !window.EventSource) return;

    if (esRef.current) esRef.current.close();
    const url = `${baseUrl}/api/v1/oauth/token-events/stream?token=${encodeURIComponent(tok)}&tenant_id=${encodeURIComponent(tenantId)}`;
    const es = new EventSource(url);
    esRef.current = es;

    es.onopen = () => { setConnected(true); setError(null); };
    es.onmessage = (msg) => {
      if (pausedRef.current) return;
      try {
        const ev: TokenEvent = JSON.parse(msg.data);
        setEvents((prev) => [ev, ...prev].slice(0, 100));
      } catch { /* ignore */ }
    };
    es.onerror = () => {
      setConnected(false);
      es.close();
      reconnectRef.current = setTimeout(() => connect(), 5000);
    };
  };

  useEffect(() => {
    connect();
    return () => { if (esRef.current) esRef.current.close(); if (reconnectRef.current) clearTimeout(reconnectRef.current); };
  }, []);

  const togglePause = () => { const np = !paused; setPaused(np); pausedRef.current = np; };
  const clearEvents = () => setEvents([]);

  const filtered = filter === "all" ? events : events.filter((e) => e.event_type === filter);
  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><KeyRound className="h-6 w-6 text-indigo-600" /> {t("auditTokenEvents.title")}</h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">Real-time token lifecycle events via SSE stream.</p>
        </div>
        <div className="flex items-center gap-2">
          <div className="flex items-center gap-1.5">
            <Radio className={`h-4 w-4 ${connected ? "text-green-500" : "text-gray-400"}`} />
            <span className="text-xs text-gray-400">{connected ? "Live" : "Disconnected"}</span>
          </div>
          <button onClick={togglePause} aria-label={paused ? "Resume stream" : "Pause stream"} aria-pressed={paused} className={`flex items-center gap-2 rounded-lg px-3 py-2 text-sm font-medium ${paused ? "bg-green-600 text-white hover:bg-green-700" : "bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30"}`}>{paused ? <><Play className="h-4 w-4" /> Resume</> : <><Pause className="h-4 w-4" /> Pause</>}</button>
          <button onClick={clearEvents} aria-label="Clear event log" className="flex items-center gap-2 rounded-lg border border-gray-300 px-3 py-2 text-sm text-gray-600 hover:bg-gray-50 dark:border-gray-600 dark:text-gray-300"><Trash2 className="h-4 w-4" /> Clear</button>
        </div>
      </div>

      {error && <div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400"><AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} aria-label="Dismiss error" className="ml-auto"><X className="h-4 w-4" /></button></div>}

      {/* Filter tabs */}
      <div className="flex gap-2">
        {["all", "issued", "refreshed", "revoked", "expired", "introspected", "exchanged"].map((f) => (
          <button key={f} onClick={() => setFilter(f)} aria-label={`Filter by ${f}`} aria-pressed={filter === f} className={`rounded-lg px-3 py-1.5 text-xs font-medium ${filter === f ? "bg-indigo-600 text-white" : "bg-gray-100 text-gray-500 dark:bg-gray-800 dark:text-gray-400"}`}>{f}</button>
        ))}
      </div>

      {/* Events feed */}
      <div className="space-y-2">
        {filtered.length === 0 ? (
          <div className={cardCls}><div className="py-12 text-center"><KeyRound className="mx-auto h-12 w-12 text-gray-300" /><p className="mt-4 text-sm text-gray-400">{paused ? "Stream paused." : "No token events yet."}</p></div></div>
        ) : (
          filtered.map((ev) => (
            <div key={ev.id} className={`${cardCls} flex items-center justify-between py-3`}>
              <div className="flex items-center gap-3">
                <span className={`inline-flex rounded-full px-2 py-1 text-xs font-medium ${eventColors[ev.event_type] || ""}`}>{ev.event_type}</span>
                <div>
                  <div className="flex items-center gap-2">
                    <span className="text-xs font-mono text-gray-400">{ev.token_type}</span>
                    <span className="text-sm text-gray-900 dark:text-white">client: {ev.client_id.slice(0, 12)}</span>
                  </div>
                  <div className="mt-0.5 flex items-center gap-3 text-xs text-gray-400">
                    <span>user: {ev.user_id.slice(0, 8)}</span>
                    {ev.ip_address && <span className="font-mono">{ev.ip_address}</span>}
                    {ev.scopes.length > 0 && <span className="text-gray-500">scopes: {ev.scopes.join(", ")}</span>}
                  </div>
                </div>
              </div>
              <span className="text-xs text-gray-400">{new Date(ev.created_at).toLocaleTimeString()}</span>
            </div>
          ))
        )}
      </div>
    </div>
  );
}
