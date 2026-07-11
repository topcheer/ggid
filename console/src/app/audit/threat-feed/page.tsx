"use client";

import React, { useState, useRef, useEffect } from "react";
import { useApi } from "@/lib/api";
import {
  Radio, Loader2, AlertCircle, X, Play, Pause, Trash2, Shield, Activity,
} from "lucide-react";

interface ThreatEvent {
  id: string;
  severity: "info" | "low" | "medium" | "high" | "critical";
  type: string;
  description: string;
  source_ip: string;
  indicators: string[];
  target: string;
  source: string;
  created_at: string;
}

const sevColors: Record<string, string> = {
  info: "text-blue-600 bg-blue-100 dark:bg-blue-900/30 dark:text-blue-400",
  low: "text-green-600 bg-green-100 dark:bg-green-900/30 dark:text-green-400",
  medium: "text-yellow-600 bg-yellow-100 dark:bg-yellow-900/30 dark:text-yellow-400",
  high: "text-orange-600 bg-orange-100 dark:bg-orange-900/30 dark:text-orange-400",
  critical: "text-red-600 bg-red-100 dark:bg-red-900/30 dark:text-red-400",
};

export default function ThreatFeedPage() {
  const { apiFetch } = useApi();
  const [threats, setThreats] = useState<ThreatEvent[]>([]);
  const [connected, setConnected] = useState(false);
  const [paused, setPaused] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [filter, setFilter] = useState<string>("all");
  const esRef = useRef<EventSource | null>(null);
  const pausedRef = useRef(false);

  const connect = () => {
    const tok = typeof window !== "undefined" ? localStorage.getItem("ggid_token") || "" : "";
    const baseUrl = typeof window !== "undefined" ? localStorage.getItem("ggid_api_base") || "" : "";
    const tenantId = typeof window !== "undefined" ? localStorage.getItem("ggid_tenant_id") || "" : "";
    if (!tok || !window.EventSource) return;
    if (esRef.current) esRef.current.close();
    const url = `${baseUrl}/api/v1/audit/threat-feed/stream?token=${encodeURIComponent(tok)}&tenant_id=${encodeURIComponent(tenantId)}`;
    const es = new EventSource(url);
    esRef.current = es;
    es.onopen = () => { setConnected(true); setError(null); };
    es.onmessage = (msg) => {
      if (pausedRef.current) return;
      try { setThreats((prev) => [JSON.parse(msg.data), ...prev].slice(0, 100)); } catch { /* ignore */ }
    };
    es.onerror = () => { setConnected(false); es.close(); setTimeout(() => connect(), 5000); };
  };

  useEffect(() => { connect(); return () => { if (esRef.current) esRef.current.close(); }; }, []);

  const togglePause = () => { const np = !paused; setPaused(np); pausedRef.current = np; };

  const filtered = filter === "all" ? threats : threats.filter((t) => t.severity === filter);
  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><Shield className="h-6 w-6 text-red-600" /> Threat Feed</h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">Real-time threat intelligence via SSE stream.</p>
        </div>
        <div className="flex items-center gap-2">
          <div className="flex items-center gap-1.5"><Radio className={`h-4 w-4 ${connected ? "text-green-500" : "text-gray-400"}`} /><span className="text-xs text-gray-400">{connected ? "Live" : "Disconnected"}</span></div>
          <button onClick={togglePause} className={`flex items-center gap-2 rounded-lg px-3 py-2 text-sm font-medium ${paused ? "bg-green-600 text-white hover:bg-green-700" : "bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30"}`}>{paused ? <><Play className="h-4 w-4" /> Resume</> : <><Pause className="h-4 w-4" /> Pause</>}</button>
          <button onClick={() => setThreats([])} className="flex items-center gap-2 rounded-lg border border-gray-300 px-3 py-2 text-sm text-gray-600 dark:border-gray-600 dark:text-gray-300"><Trash2 className="h-4 w-4" /></button>
        </div>
      </div>

      {error && <div className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400"><AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} className="ml-auto"><X className="h-4 w-4" /></button></div>}

      {/* Filter tabs */}
      <div className="flex gap-2">
        {["all", "critical", "high", "medium", "low", "info"].map((f) => (
          <button key={f} onClick={() => setFilter(f)} className={`rounded-lg px-3 py-1.5 text-xs font-medium ${filter === f ? "bg-red-600 text-white" : "bg-gray-100 text-gray-500 dark:bg-gray-800"}`}>{f}</button>
        ))}
      </div>

      {/* Stats */}
      <div className="grid grid-cols-4 gap-4">
        <div className={cardCls}><div className="text-xs font-semibold uppercase text-gray-400">Total</div><p className="mt-2 text-2xl font-bold text-gray-700 dark:text-gray-200">{threats.length}</p></div>
        <div className={cardCls}><div className="text-xs font-semibold uppercase text-gray-400">Critical</div><p className="mt-2 text-2xl font-bold text-red-600">{threats.filter((t) => t.severity === "critical").length}</p></div>
        <div className={cardCls}><div className="text-xs font-semibold uppercase text-gray-400">High</div><p className="mt-2 text-2xl font-bold text-orange-600">{threats.filter((t) => t.severity === "high").length}</p></div>
        <div className={cardCls}><div className="text-xs font-semibold uppercase text-gray-400">Filtered</div><p className="mt-2 text-2xl font-bold text-indigo-600">{filtered.length}</p></div>
      </div>

      {/* Feed */}
      <div className="space-y-2">
        {filtered.length === 0 ? (
          <div className={cardCls}><div className="py-12 text-center"><Shield className="mx-auto h-12 w-12 text-gray-300" /><p className="mt-4 text-sm text-gray-400">{paused ? "Feed paused." : "No threats detected."}</p></div></div>
        ) : filtered.map((t) => (
          <div key={t.id} className={`${cardCls} flex items-center justify-between py-3`}>
            <div className="flex items-center gap-3">
              <span className={`inline-flex rounded-full px-2 py-1 text-xs font-medium ${sevColors[t.severity] || ""}`}>{t.severity}</span>
              <div>
                <div className="flex items-center gap-2"><span className="text-sm font-medium text-gray-900 dark:text-white">{t.type}</span><span className="text-xs text-gray-400">from {t.source}</span></div>
                <div className="mt-0.5 text-xs text-gray-400">{t.description}</div>
                <div className="mt-1 flex flex-wrap items-center gap-2 text-xs text-gray-400"><span className="font-mono">IP: {t.source_ip}</span><span>Target: {t.target}</span>{t.indicators.length > 0 && <span>IOCs: {t.indicators.join(", ")}</span>}</div>
              </div>
            </div>
            <span className="text-xs text-gray-400">{new Date(t.created_at).toLocaleTimeString()}</span>
          </div>
        ))}
      </div>
    </div>
  );
}
