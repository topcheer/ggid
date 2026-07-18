"use client";
import { useState, useCallback, useEffect } from "react";
import {
  ShieldCheck, Loader2, AlertCircle, X, Search, RefreshCw,
  CheckCircle2, XCircle, Clock, ChevronRight, Activity,
  TrendingUp, Database, Zap, Filter,
} from "lucide-react";
import { authHeader } from "@/lib/auth-helpers";
import { useTranslations } from "@/lib/i18n";

const TENANT_ID = "00000000-0000-0000-0000-000000000001";

interface PDPDecision { id: string; subject: string; resource: string; action: string; decision: "allow" | "deny" | "step_up"; deny_reason?: string; risk_score: number; latency_ms: number; cache_hit: boolean; timestamp: string; }

type Tab = "log" | "analytics" | "denied";

const DEC_CFG: Record<string, { color: string; bg: string; icon: typeof CheckCircle2 }> = {
  allow: { color: "text-green-600", bg: "bg-green-100 dark:bg-green-900/30", icon: CheckCircle2 },
  deny: { color: "text-red-600", bg: "bg-red-100 dark:bg-red-900/30", icon: XCircle },
  step_up: { color: "text-yellow-600", bg: "bg-yellow-100 dark:bg-yellow-900/30", icon: ShieldCheck },
};

const SAMPLE_DECISIONS: PDPDecision[] = [
  { id: "d1", subject: "user:alice", resource: "doc:report-q4", action: "read", decision: "allow", risk_score: 12, latency_ms: 4, cache_hit: true, timestamp: new Date(Date.now() - 60000).toISOString() },
  { id: "d2", subject: "user:bob", resource: "admin:settings", action: "write", decision: "deny", deny_reason: "insufficient_role", risk_score: 45, latency_ms: 12, cache_hit: false, timestamp: new Date(Date.now() - 120000).toISOString() },
  { id: "d3", subject: "user:carol", resource: "api:/v1/users", action: "delete", decision: "step_up", risk_score: 67, latency_ms: 8, cache_hit: false, timestamp: new Date(Date.now() - 180000).toISOString() },
  { id: "d4", subject: "user:dave", resource: "doc:payroll", action: "read", decision: "deny", deny_reason: "abac_policy_violation", risk_score: 34, latency_ms: 15, cache_hit: false, timestamp: new Date(Date.now() - 300000).toISOString() },
  { id: "d5", subject: "user:eve", resource: "admin:users", action: "create", decision: "deny", deny_reason: "rbac_missing_role", risk_score: 82, latency_ms: 6, cache_hit: true, timestamp: new Date(Date.now() - 600000).toISOString() },
  { id: "d6", subject: "user:frank", resource: "api:/v1/exports", action: "execute", decision: "allow", risk_score: 8, latency_ms: 3, cache_hit: true, timestamp: new Date(Date.now() - 900000).toISOString() },
  { id: "d7", subject: "user:alice", resource: "admin:orgs", action: "write", decision: "step_up", risk_score: 58, latency_ms: 11, cache_hit: false, timestamp: new Date(Date.now() - 1200000).toISOString() },
  { id: "d8", subject: "user:bob", resource: "secret:vault", action: "read", decision: "deny", deny_reason: "jit_required", risk_score: 91, latency_ms: 9, cache_hit: false, timestamp: new Date(Date.now() - 1800000).toISOString() },
];

export default function PDPDecisionsPage() {
  const t = useTranslations();
  const [tab, setTab] = useState<Tab>("log");
  const [decisions, setDecisions] = useState<PDPDecision[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [search, setSearch] = useState("");

  const h = { ...authHeader(), "X-Tenant-ID": TENANT_ID };
  const card = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  const loadData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/v1/policy/decisions", { headers: h }).catch(() => null);
      if (res?.ok) { const d = await res.json(); setDecisions(d.decisions || d.items || []); }
      else setDecisions(SAMPLE_DECISIONS);
    } catch { setError(t("pdp.loadError")); }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { loadData(); }, [loadData]);

  const filtered = search ? decisions.filter(d => d.subject.includes(search) || d.resource.includes(search) || d.action.includes(search)) : decisions;
  const allowCount = decisions.filter(d => d.decision === "allow").length;
  const denyCount = decisions.filter(d => d.decision === "deny").length;
  const stepUpCount = decisions.filter(d => d.decision === "step_up").length;
  const avgLatency = decisions.length > 0 ? Math.round(decisions.reduce((a, d) => a + d.latency_ms, 0) / decisions.length) : 0;
  const cacheHitRate = decisions.length > 0 ? Math.round(decisions.filter(d => d.cache_hit).length / decisions.length * 100) : 0;
  const deniedDecisions = decisions.filter(d => d.decision === "deny");
  const total = decisions.length || 1;

  return (
    <div className="space-y-6">
      <div><h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><ShieldCheck className="h-6 w-6 text-emerald-500" /> {t("pdp.title")}</h1><p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{t("pdp.subtitle")}</p></div>

      {error && (<div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400"><AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} aria-label="Dismiss" className="ml-auto"><X className="h-4 w-4" /></button></div>)}

      <div className="flex gap-1 border-b border-gray-200 dark:border-gray-700 overflow-x-auto">
        {([["log", t("pdp.decisionLog"), Activity], ["analytics", t("pdp.analytics"), TrendingUp], ["denied", `${t("pdp.deniedRequests")} (${denyCount})`, XCircle]] as const).map(([id, label, Icon]) => (
          <button key={id} onClick={() => setTab(id as Tab)} aria-pressed={tab === id} className={`flex items-center gap-1.5 border-b-2 px-4 py-2 text-sm font-medium transition whitespace-nowrap ${tab === id ? "border-emerald-600 text-emerald-600 dark:text-emerald-400" : "border-transparent text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"}`}><Icon className="h-4 w-4" /> {label}</button>
        ))}
      </div>

      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-emerald-500" /></div> : (<>

      {/* LOG */}
      {tab === "log" && (
        <div>
          <div className="mb-4"><div className="relative max-w-xs"><Search className="absolute left-3 top-2.5 h-4 w-4 text-gray-400" /><input type="text" value={search} onChange={e => setSearch(e.target.value)} placeholder={t("pdp.search")} className="w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 pl-9 pr-3 py-1.5 text-sm" /></div></div>
          <div className="overflow-x-auto"><table className="w-full text-sm">
            <thead className="bg-gray-50 dark:bg-gray-800/50"><tr><th className="px-3 py-2 text-left text-xs text-gray-400">{t("pdp.subject")}</th><th className="px-3 py-2 text-left text-xs text-gray-400">{t("pdp.resource")}</th><th className="px-3 py-2 text-center text-xs text-gray-400">{t("pdp.action")}</th><th className="px-3 py-2 text-center text-xs text-gray-400">{t("pdp.decision")}</th><th className="px-3 py-2 text-center text-xs text-gray-400">{t("pdp.risk")}</th><th className="px-3 py-2 text-center text-xs text-gray-400">{t("pdp.latency")}</th><th className="px-3 py-2 text-center text-xs text-gray-400">{t("pdp.cache")}</th><th className="px-3 py-2 text-left text-xs text-gray-400">{t("pdp.time")}</th></tr></thead>
            <tbody className="divide-y dark:divide-gray-800">{filtered.slice(0, 50).map(d => { const cfg = DEC_CFG[d.decision]; const DIcon = cfg.icon; return (
              <tr key={d.id} className="hover:bg-gray-50 dark:hover:bg-gray-900/30"><td className="px-3 py-3 text-xs font-mono">{d.subject}</td><td className="px-3 py-3 text-xs font-mono text-gray-500">{d.resource}</td><td className="px-3 py-3 text-center"><code className="text-xs font-mono">{d.action}</code></td><td className="px-3 py-3 text-center"><span className={`inline-flex items-center gap-1 px-1.5 py-0.5 rounded text-xs font-medium ${cfg.bg} ${cfg.color}`}><DIcon className="h-3 w-3" /> {d.decision}</span></td><td className="px-3 py-3 text-center"><span className={`text-xs font-mono ${d.risk_score >= 60 ? "text-red-600" : d.risk_score >= 30 ? "text-yellow-600" : "text-green-600"}`}>{d.risk_score}</span></td><td className="px-3 py-3 text-center text-xs font-mono">{d.latency_ms}ms</td><td className="px-3 py-3 text-center">{d.cache_hit ? <CheckCircle2 className="mx-auto h-3.5 w-3.5 text-green-500" /> : <span className="text-xs text-gray-300">—</span>}</td><td className="px-3 py-3 text-xs text-gray-400">{new Date(d.timestamp).toLocaleTimeString()}</td></tr>
            );})}</tbody>
          </table></div>
        </div>
      )}

      {/* ANALYTICS */}
      {tab === "analytics" && (
        <div className="space-y-6">
          <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
            <div className={card + " text-center"}><Activity className="mx-auto h-5 w-5 text-emerald-400" /><p className="mt-2 text-2xl font-bold">{decisions.length}</p><p className="text-xs text-gray-400">{t("pdp.totalDecisions")}</p></div>
            <div className={card + " text-center"}><Zap className="mx-auto h-5 w-5 text-blue-400" /><p className="mt-2 text-2xl font-bold">{avgLatency}ms</p><p className="text-xs text-gray-400">{t("pdp.avgLatency")}</p></div>
            <div className={card + " text-center"}><Database className="mx-auto h-5 w-5 text-purple-400" /><p className="mt-2 text-2xl font-bold">{cacheHitRate}%</p><p className="text-xs text-gray-400">{t("pdp.cacheHitRate")}</p></div>
            <div className={card + " text-center"}><TrendingUp className="mx-auto h-5 w-5 text-green-400" /><p className="mt-2 text-2xl font-bold text-green-600">{Math.round(allowCount / total * 100)}%</p><p className="text-xs text-gray-400">{t("pdp.allowRate")}</p></div>
          </div>
          <div className={card}>
            <h3 className="mb-4 text-sm font-semibold uppercase text-gray-400">{t("pdp.decisionBreakdown")}</h3>
            <div className="flex h-8 overflow-hidden rounded-full">
              <div className="flex items-center justify-center text-xs font-bold text-white bg-green-500" style={{ width: `${allowCount / total * 100}%` }}>{allowCount > 0 && `${Math.round(allowCount / total * 100)}%`}</div>
              <div className="flex items-center justify-center text-xs font-bold text-white bg-yellow-500" style={{ width: `${stepUpCount / total * 100}%` }}>{stepUpCount > 0 && `${Math.round(stepUpCount / total * 100)}%`}</div>
              <div className="flex items-center justify-center text-xs font-bold text-white bg-red-500" style={{ width: `${denyCount / total * 100}%` }}>{denyCount > 0 && `${Math.round(denyCount / total * 100)}%`}</div>
            </div>
            <div className="mt-3 flex gap-4 text-xs"><span className="flex items-center gap-1"><span className="h-2.5 w-2.5 rounded-full bg-green-500" /> Allow ({allowCount})</span><span className="flex items-center gap-1"><span className="h-2.5 w-2.5 rounded-full bg-yellow-500" /> Step-up ({stepUpCount})</span><span className="flex items-center gap-1"><span className="h-2.5 w-2.5 rounded-full bg-red-500" /> Deny ({denyCount})</span></div>
          </div>
        </div>
      )}

      {/* DENIED */}
      {tab === "denied" && (
        <div className="space-y-2">
          {deniedDecisions.length === 0 ? <div className={card}><div className="py-12 text-center"><CheckCircle2 className="mx-auto h-12 w-12 text-green-300" /><p className="mt-4 text-sm text-gray-400">{t("pdp.noDenied")}</p></div></div> :
          deniedDecisions.map(d => (
            <div key={d.id} className={`${card} flex items-center justify-between !p-3`}>
              <div className="flex items-center gap-3"><div className="flex h-8 w-8 items-center justify-center rounded-lg bg-red-100 dark:bg-red-900/30"><XCircle className="h-4 w-4 text-red-500" /></div><div><div className="flex items-center gap-2"><span className="text-xs font-mono">{d.subject}</span><ChevronRight className="h-3 w-3 text-gray-300" /><code className="text-xs font-mono text-gray-500">{d.resource}</code></div><p className="text-xs text-gray-400">{t("pdp.reason")}: <code className="font-mono text-red-500">{d.deny_reason}</code> · {t("pdp.risk")}: {d.risk_score} · {new Date(d.timestamp).toLocaleString()}</p></div></div>
              <span className="px-1.5 py-0.5 rounded text-xs bg-red-100 dark:bg-red-900/30 text-red-600">{d.action}</span>
            </div>
          ))}
        </div>
      )}

      </>)}
    </div>
  );
}
