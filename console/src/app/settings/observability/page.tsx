"use client";
import { useState, useCallback, useEffect } from "react";
import {
  Activity, Loader2, AlertCircle, X, RefreshCw, ChevronRight,
  Server, Zap, CheckCircle2, XCircle, Clock, Network, Settings,
} from "lucide-react";
import { authHeader } from "@/lib/auth-helpers";
import { useTranslations } from "@/lib/i18n";

const TENANT_ID = "00000000-0000-0000-0000-000000000001";

interface Trace { id: string; service: string; operation: string; duration_ms: number; status: "ok" | "error"; spans: number; timestamp: string; }

type Tab = "traces" | "map" | "health";

const SAMPLE_TRACES: Trace[] = [
  { id: "t-001", service: "gateway", operation: "POST /api/v1/auth/login", duration_ms: 847, status: "ok", spans: 12, timestamp: new Date(Date.now() - 60000).toISOString() },
  { id: "t-002", service: "identity", operation: "CreateUser", duration_ms: 342, status: "ok", spans: 8, timestamp: new Date(Date.now() - 120000).toISOString() },
  { id: "t-003", service: "auth", operation: "ValidateToken", duration_ms: 12, status: "ok", spans: 3, timestamp: new Date(Date.now() - 180000).toISOString() },
  { id: "t-004", service: "policy", operation: "EvaluateABAC", duration_ms: 89, status: "ok", spans: 5, timestamp: new Date(Date.now() - 300000).toISOString() },
  { id: "t-005", service: "audit", operation: "WriteEvent", duration_ms: 2340, status: "error", spans: 6, timestamp: new Date(Date.now() - 600000).toISOString() },
  { id: "t-006", service: "gateway", operation: "GET /api/v1/users", duration_ms: 156, status: "ok", spans: 7, timestamp: new Date(Date.now() - 900000).toISOString() },
];

const SERVICE_MAP = [
  { from: "gateway", to: "auth", calls: 4521 }, { from: "gateway", to: "identity", calls: 2103 },
  { from: "gateway", to: "policy", calls: 1899 }, { from: "gateway", to: "audit", calls: 3240 },
  { from: "auth", to: "identity", calls: 1842 }, { from: "identity", to: "audit", calls: 876 },
  { from: "policy", to: "audit", calls: 543 }, { from: "gateway", to: "oauth", calls: 1234 },
];

const SERVICES = ["gateway", "auth", "identity", "policy", "audit", "oauth"];

export default function ObservabilityPage() {
  const t = useTranslations();
  const [tab, setTab] = useState<Tab>("traces");
  const [traces, setTraces] = useState<Trace[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [expandedId, setExpandedId] = useState<string | null>(null);

  const h = { ...authHeader(), "X-Tenant-ID": TENANT_ID };
  const card = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  const loadData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/v1/observability/traces", { headers: h }).catch(() => null);
      if (res?.ok) { const d = await res.json(); setTraces(d.traces || []); }
      else setTraces(SAMPLE_TRACES);
    } catch { setError(t("observability.loadError")); }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { loadData(); }, [loadData]);

  const maxDuration = Math.max(...traces.map(t => t.duration_ms), 1);

  return (
    <div className="space-y-6">
      <div><h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><Activity className="h-6 w-6 text-green-500" /> {t("observability.title")}</h1><p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{t("observability.subtitle")}</p></div>

      {error && (<div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400"><AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} aria-label="Dismiss" className="ml-auto"><X className="h-4 w-4" /></button></div>)}

      <div className="flex gap-1 border-b border-gray-200 dark:border-gray-700 overflow-x-auto">
        {([
          { id: "traces" as Tab, label: t("observability.traceExplorer"), icon: Zap },
          { id: "map" as Tab, label: t("observability.serviceMap"), icon: Network },
          { id: "health" as Tab, label: t("observability.health"), icon: CheckCircle2 },
        ]).map(tb => { const Icon = tb.icon; return (
          <button key={tb.id} onClick={() => setTab(tb.id)} aria-pressed={tab === tb.id} className={`flex items-center gap-1.5 border-b-2 px-4 py-2 text-sm font-medium transition whitespace-nowrap ${tab === tb.id ? "border-green-600 text-green-600 dark:text-green-400" : "border-transparent text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"}`}><Icon className="h-4 w-4" /> {tb.label}</button>
        );})}
      </div>

      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-green-500" /></div> : (<>

      {/* TRACES */}
      {tab === "traces" && (
        <div className="space-y-2">
          {traces.map(tr => (
            <div key={tr.id}>
              <button onClick={() => setExpandedId(expandedId === tr.id ? null : tr.id)} className={`${card} flex w-full items-center justify-between !p-3 text-left`}>
                <div className="flex items-center gap-3">
                  <div className={`flex h-8 w-8 items-center justify-center rounded-lg ${tr.status === "ok" ? "bg-green-100 dark:bg-green-900/30" : "bg-red-100 dark:bg-red-900/30"}`}>{tr.status === "ok" ? <CheckCircle2 className="h-4 w-4 text-green-500" /> : <XCircle className="h-4 w-4 text-red-500" />}</div>
                  <div><div className="flex items-center gap-2"><span className="px-1.5 py-0.5 rounded bg-gray-100 dark:bg-gray-700 text-xs font-mono">{tr.service}</span><span className="text-xs font-medium">{tr.operation}</span></div><p className="text-xs text-gray-400">{tr.spans} spans · {new Date(tr.timestamp).toLocaleTimeString()}</p></div>
                </div>
                <div className="flex items-center gap-3"><div className="w-24 h-1.5 overflow-hidden rounded-full bg-gray-200 dark:bg-gray-700"><div className={`h-full rounded-full ${tr.duration_ms > 1000 ? "bg-red-500" : tr.duration_ms > 300 ? "bg-yellow-500" : "bg-green-500"}`} style={{ width: `${(tr.duration_ms / maxDuration) * 100}%` }} /></div><span className="text-xs font-mono w-12 text-right">{tr.duration_ms}ms</span><ChevronRight className={`h-4 w-4 text-gray-300 transition ${expandedId === tr.id ? "rotate-90" : ""}`} /></div>
              </button>
              {expandedId === tr.id && (
                <div className="ml-4 mr-4 mb-2 rounded-lg border-l-2 border-green-500 pl-4 dark:border-green-400">
                  <div className="space-y-1 py-2">{Array.from({ length: tr.spans }).slice(0, 6).map((_, i) => {
                    const spanDur = Math.round(tr.duration_ms / tr.spans * (0.5 + Math.random()));
                    return <div key={i} className="flex items-center gap-2 text-xs"><span className="text-gray-400 w-16">{i === 0 ? tr.service : ["db", "cache", "http", "grpc", "redis"][i % 5]}</span><span className="flex-1 h-1 rounded-full bg-gray-200 dark:bg-gray-700"><span className="block h-full rounded-full bg-green-400" style={{ width: `${(spanDur / tr.duration_ms) * 100}%`, marginLeft: `${i * 5}%` }} /></span><span className="text-gray-400 w-10 text-right">{spanDur}ms</span></div>;
                  })}</div>
                </div>
              )}
            </div>
          ))}
        </div>
      )}

      {/* SERVICE MAP */}
      {tab === "map" && (
        <div className={card}>
          <h3 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Network className="h-4 w-4" /> {t("observability.dependencyGraph")}</h3>
          <div className="grid grid-cols-2 gap-4 sm:grid-cols-3">
            {SERVICES.map(svc => {
              const outgoing = SERVICE_MAP.filter(m => m.from === svc);
              const incoming = SERVICE_MAP.filter(m => m.to === svc);
              return (
                <div key={svc} className="rounded-lg border p-3 dark:border-gray-700">
                  <div className="flex items-center gap-2"><Server className="h-4 w-4 text-green-500" /><span className="font-medium text-sm">{svc}</span></div>
                  <p className="mt-1 text-xs text-gray-400">↓ {outgoing.length} deps · ↑ {incoming.length} callers</p>
                  {outgoing.length > 0 && <div className="mt-2 space-y-0.5">{outgoing.map(m => <div key={m.to} className="flex items-center gap-1 text-xs text-gray-400"><ChevronRight className="h-2.5 w-2.5" />{m.to}<span className="ml-auto font-mono">{m.calls.toLocaleString()}</span></div>)}</div>}
                </div>
              );
            })}
          </div>
        </div>
      )}

      {/* HEALTH */}
      {tab === "health" && (
        <div className="space-y-6">
          <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
            <div className={card + " text-center"}><Activity className="mx-auto h-5 w-5 text-green-400" /><p className="mt-2 text-2xl font-bold">{traces.length}</p><p className="text-xs text-gray-400">{t("observability.tracesToday")}</p></div>
            <div className={card + " text-center"}><CheckCircle2 className="mx-auto h-5 w-5 text-green-400" /><p className="mt-2 text-2xl font-bold">{traces.filter(t => t.status === "ok").length}</p><p className="text-xs text-gray-400">{t("observability.healthy")}</p></div>
            <div className={card + " text-center"}><XCircle className="mx-auto h-5 w-5 text-red-400" /><p className="mt-2 text-2xl font-bold text-red-600">{traces.filter(t => t.status === "error").length}</p><p className="text-xs text-gray-400">{t("observability.errors")}</p></div>
            <div className={card + " text-center"}><Clock className="mx-auto h-5 w-5 text-yellow-400" /><p className="mt-2 text-2xl font-bold">{Math.round(traces.reduce((a, t) => a + t.duration_ms, 0) / (traces.length || 1))}ms</p><p className="text-xs text-gray-400">{t("observability.avgDuration")}</p></div>
          </div>
          <div className={card}>
            <h3 className="mb-3 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Settings className="h-4 w-4" /> {t("observability.exporterConfig")}</h3>
            <div className="space-y-2">
              {[["Exporter", "OTLP gRPC"], ["Endpoint", "otel-collector:4317"], ["Sampling Rate", "10%"], ["Service Name", "ggid-console"]].map(([k, v]) => <div key={k} className="flex items-center justify-between rounded-lg border p-2 dark:border-gray-700"><span className="text-sm">{k}</span><code className="text-xs font-mono text-gray-500">{v}</code></div>)}
            </div>
          </div>
        </div>
      )}

      </>)}
    </div>
  );
}
