"use client";
import { useState, useCallback, useEffect } from "react";
import {
  Clock, Loader2, AlertCircle, X, Search, RefreshCw, ChevronRight,
  LogIn, Shield, KeyRound, FileText, Activity, Smartphone, Globe,
  CheckCircle2, XCircle, AlertTriangle, Filter,
} from "lucide-react";
import { authHeader } from "@/lib/auth-helpers";
import { useTranslations } from "@/lib/i18n";

const TENANT_ID = "00000000-0000-0000-0000-000000000001";

interface AuditEvent { id: string; action: string; actor: string; resource: string; ip: string; timestamp: string; result: "success" | "denied" | "error"; details: Record<string, unknown>; }

type EventType = "all" | "login" | "mfa" | "token" | "policy" | "risk" | "session";

const EVENT_ICONS: Record<string, typeof LogIn> = {
  login: LogIn, mfa: Shield, token: KeyRound, policy: FileText, risk: Activity, session: Smartphone,
};
const RESULT_CFG: Record<string, string> = {
  success: "text-green-600 bg-green-100 dark:bg-green-900/30",
  denied: "text-red-600 bg-red-100 dark:bg-red-900/30",
  error: "text-orange-600 bg-orange-100 dark:bg-orange-900/30",
};

function classifyEvent(action: string): EventType {
  if (action.includes("login") || action.includes("auth")) return "login";
  if (action.includes("mfa") || action.includes("otp") || action.includes("webauthn")) return "mfa";
  if (action.includes("token") || action.includes("oauth")) return "token";
  if (action.includes("policy") || action.includes("decision")) return "policy";
  if (action.includes("risk")) return "risk";
  return "session";
}

export default function UserTimelinePage() {
  const t = useTranslations();
  const [events, setEvents] = useState<AuditEvent[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [searchUser, setSearchUser] = useState("");
  const [filterType, setFilterType] = useState<EventType>("all");
  const [expandedId, setExpandedId] = useState<string | null>(null);

  const h = { ...authHeader(), "X-Tenant-ID": TENANT_ID };
  const card = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  const loadData = useCallback(async () => {
    setLoading(true);
    try {
      const params = new URLSearchParams({ page_size: "100", actor: searchUser });
      const res = await fetch(`/api/v1/audit/events?${params}`, { headers: h }).catch(() => null);
      if (res?.ok) { const d = await res.json(); setEvents(d.events || d.items || []); }
    } catch { setError(t("userTimeline.loadError")); }
    finally { setLoading(false); }
  }, [searchUser]);

  useEffect(() => { loadData(); }, [loadData]);

  const filtered = events.filter(e => {
    const evType = classifyEvent(e.action);
    if (filterType !== "all" && evType !== filterType) return false;
    return true;
  });

  const eventTypeCounts = events.reduce((acc, e) => { const type = classifyEvent(e.action); acc[type] = (acc[type] || 0) + 1; return acc; }, {} as Record<string, number>);

  return (
    <div className="space-y-6">
      <div><h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><Clock className="h-6 w-6 text-blue-500" /> {t("userTimeline.title")}</h1><p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{t("userTimeline.subtitle")}</p></div>

      {error && (<div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400"><AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} aria-label="Dismiss" className="ml-auto"><X className="h-4 w-4" /></button></div>)}

      {/* Filters */}
      <div className="flex items-center gap-3 flex-wrap">
        <div className="relative flex-1 min-w-xs max-w-xs"><Search className="absolute left-3 top-2.5 h-4 w-4 text-gray-400" /><input type="text" value={searchUser} onChange={e => setSearchUser(e.target.value)} placeholder={t("userTimeline.searchUser")} className="w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 pl-9 pr-3 py-1.5 text-sm" /></div>
        <div className="flex items-center gap-1">
          {(["all", "login", "mfa", "token", "policy", "risk", "session"] as EventType[]).map(typ => (
            <button key={typ} onClick={() => setFilterType(typ)} aria-pressed={filterType === typ} className={`rounded-lg px-2.5 py-1 text-xs font-medium transition ${filterType === typ ? "bg-blue-600 text-white" : "bg-gray-100 dark:bg-gray-800 text-gray-500"}`}>
              {typ === "all" ? t("userTimeline.allTypes") : typ}{typ !== "all" && eventTypeCounts[typ] ? ` (${eventTypeCounts[typ]})` : ""}
            </button>
          ))}
        </div>
        <button onClick={loadData} aria-label="Refresh" className="rounded-lg border border-gray-300 p-1.5 dark:border-gray-700"><RefreshCw className="h-3.5 w-3.5" /></button>
      </div>

      {/* Timeline */}
      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-blue-500" /></div> : filtered.length === 0 ? (
        <div className={card}><div className="py-12 text-center"><Clock className="mx-auto h-12 w-12 text-gray-300" /><p className="mt-4 text-sm text-gray-400">{t("userTimeline.noEvents")}</p></div></div>
      ) : (
        <div className="relative">
          <div className="absolute left-4 top-0 bottom-0 w-px bg-gray-200 dark:bg-gray-700" />
          <div className="space-y-1">{filtered.slice(0, 50).map(ev => {
            const evType = classifyEvent(ev.action);
            const EIcon = EVENT_ICONS[evType] || Activity;
            const resultClass = RESULT_CFG[ev.result] || RESULT_CFG.success;
            const isExpanded = expandedId === ev.id;
            return (
              <div key={ev.id} className="relative flex items-start gap-4 pl-0">
                <button onClick={() => setExpandedId(isExpanded ? null : ev.id)} className="relative z-10 flex h-8 w-8 items-center justify-center rounded-full bg-gray-100 dark:bg-gray-800 shrink-0 hover:bg-gray-200 dark:hover:bg-gray-700" aria-label="Toggle details">
                  <EIcon className="h-4 w-4 text-blue-500" />
                </button>
                <div className="flex-1 pb-2">
                  <button onClick={() => setExpandedId(isExpanded ? null : ev.id)} className="w-full text-left">
                    <div className="flex items-center gap-2 flex-wrap">
                      <span className="text-sm font-medium">{ev.action}</span>
                      <span className={`px-1.5 py-0.5 rounded text-xs font-medium ${resultClass}`}>{ev.result}</span>
                      {ev.resource && <code className="text-xs font-mono text-gray-400">{ev.resource}</code>}
                    </div>
                    <p className="text-xs text-gray-400 mt-0.5">{ev.actor} · <Globe className="inline h-2.5 w-2.5" /> {ev.ip} · {new Date(ev.timestamp).toLocaleString()}</p>
                  </button>
                  {isExpanded && (
                    <div className="mt-2 ml-2 rounded-lg border-l-2 border-blue-400 pl-3 dark:border-blue-500">
                      <div className="space-y-1">
                        {Object.entries(ev.details || {}).slice(0, 8).map(([k, v]) => (
                          <div key={k} className="flex items-center gap-2 text-xs"><span className="text-gray-400">{k}:</span><code className="font-mono text-gray-500">{typeof v === "object" ? JSON.stringify(v) : String(v)}</code></div>
                        ))}
                        {Object.keys(ev.details || {}).length === 0 && <p className="text-xs text-gray-400">{t("userTimeline.noDetails")}</p>}
                      </div>
                    </div>
                  )}
                </div>
              </div>
            );
          })}</div>
        </div>
      )}
    </div>
  );
}
