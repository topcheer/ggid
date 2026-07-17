"use client";
import { useState, useCallback, useEffect } from "react";
import {
  Bell, Loader2, AlertCircle, X, RefreshCw, Plus, Check,
  AlertTriangle, Clock, Ban, ExternalLink, Volume2, VolumeX,
} from "lucide-react";
import { authHeader } from "@/lib/auth-helpers";
import { useTranslations } from "@/lib/i18n";

const TENANT_ID = "00000000-0000-0000-0000-000000000001";

interface AlertEvent { id: string; name: string; severity: "critical" | "warning" | "info"; duration: string; value: string; threshold: string; runbook: string; }
interface AlertRule { id: string; name: string; metric: string; threshold: string; severity: string; enabled: boolean; }
interface Silence { id: string; matchers: string; starts: string; ends: string; created_by: string; reason: string; }

type Tab = "active" | "rules" | "silences";

const SEV_CFG: Record<string, { color: string; bg: string }> = {
  critical: { color: "text-red-600", bg: "bg-red-100 dark:bg-red-900/30" },
  warning: { color: "text-yellow-600", bg: "bg-yellow-100 dark:bg-yellow-900/30" },
  info: { color: "text-blue-600", bg: "bg-blue-100 dark:bg-blue-900/30" },
};

const SAMPLE_ALERTS: AlertEvent[] = [
  { id: "a1", name: "HighErrorRate", severity: "critical", duration: "12m", value: "5.2%", threshold: "<1%", runbook: "https://docs.ggid.dev/runbooks/high-error-rate" },
  { id: "a2", name: "LatencyP99High", severity: "warning", duration: "8m", value: "2.3s", threshold: "<1s", runbook: "https://docs.ggid.dev/runbooks/latency" },
  { id: "a3", name: "DBConnectionsHigh", severity: "warning", duration: "23m", value: "187", threshold: "<150", runbook: "https://docs.ggid.dev/runbooks/db-pool" },
  { id: "a4", name: "MFAFailRateHigh", severity: "critical", duration: "5m", value: "14.2%", threshold: "<5%", runbook: "https://docs.ggid.dev/runbooks/mfa" },
];

const SAMPLE_RULES: AlertRule[] = [
  { id: "r1", name: "HighErrorRate", metric: "rate(http_requests_total{status=~\"5..\"}[5m])", threshold: "> 0.01", severity: "critical", enabled: true },
  { id: "r2", name: "LatencyP99High", metric: "histogram_quantile(0.99, http_duration_seconds)", threshold: "> 1.0", severity: "warning", enabled: true },
  { id: "r3", name: "DBConnectionsHigh", metric: "pg_connections_active", threshold: "> 150", severity: "warning", enabled: true },
  { id: "r4", name: "MFAFailRateHigh", metric: "rate(mfa_failures_total[5m])", threshold: "> 0.05", severity: "critical", enabled: true },
  { id: "r5", name: "DiskUsageHigh", metric: "disk_usage_percent", threshold: "> 85", severity: "warning", enabled: true },
  { id: "r6", name: "MemoryUsageHigh", metric: "container_memory_usage_bytes", threshold: "> 0.9", severity: "warning", enabled: true },
  { id: "r7", name: "CPUHigh", metric: "rate(container_cpu_usage_seconds_total[5m])", threshold: "> 0.8", severity: "warning", enabled: true },
];

const SAMPLE_SILENCES: Silence[] = [
  { id: "s1", matchers: "service:auth, alert:HighErrorRate", starts: "2025-01-15T02:00:00Z", ends: "2025-01-15T04:00:00Z", created_by: "ops-team", reason: "Scheduled maintenance window" },
];

export default function AlertsPage() {
  const t = useTranslations();
  const [tab, setTab] = useState<Tab>("active");
  const [rules, setRules] = useState(SAMPLE_RULES);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showSilenceForm, setShowSilenceForm] = useState(false);
  const [silenceReason, setSilenceReason] = useState("");
  const [silenceDuration, setSilenceDuration] = useState(60);

  const h = { ...authHeader(), "X-Tenant-ID": TENANT_ID };
  const card = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  useEffect(() => { setLoading(false); }, []);

  const toggleRule = (id: string) => setRules(prev => prev.map(r => r.id === id ? { ...r, enabled: !r.enabled } : r));

  return (
    <div className="space-y-6">
      <div><h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><Bell className="h-6 w-6 text-red-500" /> {t("alerts.title")}</h1><p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{t("alerts.subtitle")}</p></div>

      {error && (<div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400"><AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} aria-label="Dismiss" className="ml-auto"><X className="h-4 w-4" /></button></div>)}

      <div className="flex gap-1 border-b border-gray-200 dark:border-gray-700 overflow-x-auto">
        {([
          { id: "active" as Tab, label: `${t("alerts.activeAlerts")} (${SAMPLE_ALERTS.length})`, icon: AlertTriangle },
          { id: "rules" as Tab, label: t("alerts.alertRules"), icon: Bell },
          { id: "silences" as Tab, label: t("alerts.silences"), icon: VolumeX },
        ]).map(tb => { const Icon = tb.icon; return (
          <button key={tb.id} onClick={() => setTab(tb.id)} aria-pressed={tab === tb.id} className={`flex items-center gap-1.5 border-b-2 px-4 py-2 text-sm font-medium transition whitespace-nowrap ${tab === tb.id ? "border-red-600 text-red-600 dark:text-red-400" : "border-transparent text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"}`}><Icon className="h-4 w-4" /> {tb.label}</button>
        );})}
      </div>

      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-red-500" /></div> : (<>

      {/* ACTIVE */}
      {tab === "active" && (
        <div className="space-y-2">
          {SAMPLE_ALERTS.map(a => { const cfg = SEV_CFG[a.severity]; return (
            <div key={a.id} className={`${card} flex items-center justify-between ${a.severity === "critical" ? "border-red-200 dark:border-red-800" : ""}`}>
              <div className="flex items-center gap-3"><div className={`flex h-9 w-9 items-center justify-center rounded-lg ${cfg.bg}`}><AlertTriangle className={`h-4 w-4 ${cfg.color}`} /></div><div><div className="flex items-center gap-2"><span className="font-medium text-sm">{a.name}</span><span className={`px-1.5 py-0.5 rounded text-xs font-medium ${cfg.bg} ${cfg.color}`}>{a.severity}</span><span className="flex items-center gap-1 text-xs text-gray-400"><Clock className="h-3 w-3" />{a.duration}</span></div><p className="text-xs text-gray-400">{t("alerts.value")}: <span className="font-mono">{a.value}</span> / {t("alerts.threshold")}: <span className="font-mono">{a.threshold}</span></p></div></div>
              <a href={a.runbook} target="_blank" rel="noopener" aria-label="Runbook" className="flex items-center gap-1 rounded-lg border border-gray-300 px-2 py-1 text-xs dark:border-gray-700"><ExternalLink className="h-3 w-3" /> {t("alerts.runbook")}</a>
            </div>
          );})}
        </div>
      )}

      {/* RULES */}
      {tab === "rules" && (
        <div className="overflow-x-auto"><table className="w-full text-sm">
          <thead className="bg-gray-50 dark:bg-gray-800/50"><tr><th className="px-3 py-2 text-left text-xs text-gray-400">{t("alerts.name")}</th><th className="px-3 py-2 text-left text-xs text-gray-400">{t("alerts.metric")}</th><th className="px-3 py-2 text-center text-xs text-gray-400">{t("alerts.severity")}</th><th className="px-3 py-2 text-center text-xs text-gray-400">{t("alerts.enabled")}</th></tr></thead>
          <tbody className="divide-y dark:divide-gray-800">{rules.map(r => { const cfg = SEV_CFG[r.severity] || SEV_CFG.warning; return (
            <tr key={r.id} className="hover:bg-gray-50 dark:hover:bg-gray-900/30"><td className="px-3 py-3 text-xs font-medium">{r.name}</td><td className="px-3 py-3"><code className="text-xs font-mono text-gray-500 truncate block max-w-xs">{r.metric}</code></td><td className="px-3 py-3 text-center"><span className={`px-1.5 py-0.5 rounded text-xs font-medium ${cfg.bg} ${cfg.color}`}>{r.severity}</span></td><td className="px-3 py-3 text-center"><button onClick={() => toggleRule(r.id)} aria-pressed={r.enabled} aria-label={"Toggle " + r.name} className={`relative h-5 w-9 rounded-full transition ${r.enabled ? "bg-green-500" : "bg-gray-300 dark:bg-gray-700"}`}><span className={`absolute top-0.5 h-4 w-4 rounded-full bg-white transition ${r.enabled ? "left-4" : "left-0.5"}`} /></button></td></tr>
          );})}</tbody>
        </table></div>
      )}

      {/* SILENCES */}
      {tab === "silences" && (
        <div>
          <div className="mb-4"><button onClick={() => setShowSilenceForm(true)} className="flex items-center gap-1 rounded-lg bg-red-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-red-700"><Plus className="h-3 w-3" /> {t("alerts.createSilence")}</button></div>
          {SAMPLE_SILENCES.length === 0 ? <div className={card}><div className="py-8 text-center"><Volume2 className="mx-auto h-10 w-10 text-gray-300" /><p className="mt-3 text-sm text-gray-400">{t("alerts.noSilences")}</p></div></div> : (
            <div className="space-y-2">{SAMPLE_SILENCES.map(s => (
              <div key={s.id} className={`${card} flex items-center justify-between !p-3`}>
                <div className="flex items-center gap-3"><div className="flex h-8 w-8 items-center justify-center rounded-lg bg-gray-100 dark:bg-gray-700"><VolumeX className="h-4 w-4 text-gray-400" /></div><div><span className="text-sm font-medium">{s.reason}</span><p className="text-xs text-gray-400">{s.matchers}</p><p className="text-xs text-gray-400">{new Date(s.starts).toLocaleString()} → {new Date(s.ends).toLocaleString()} · {s.created_by}</p></div></div>
                <span className="px-1.5 py-0.5 rounded text-xs bg-gray-100 dark:bg-gray-700">{t("alerts.active2")}</span>
              </div>
            ))}</div>
          )}
        </div>
      )}

      </>)}

      {showSilenceForm && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setShowSilenceForm(false)}>
          <div role="dialog" aria-modal="true" className="mx-4 w-full max-w-md rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={e => e.stopPropagation()}>
            <h3 className="flex items-center gap-2 text-lg font-semibold text-gray-900 dark:text-white"><VolumeX className="h-5 w-5 text-gray-500" /> {t("alerts.createSilence")}</h3>
            <div className="mt-4 space-y-3">
              <div><label className="text-sm font-medium">{t("alerts.reason")}</label><input type="text" value={silenceReason} onChange={e => setSilenceReason(e.target.value)} placeholder="Maintenance window" className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm" autoFocus /></div>
              <div><label className="text-sm font-medium">{t("alerts.duration2")} ({t("alerts.minutes")})</label><div className="mt-1 flex gap-2">{[30, 60, 120, 240].map(d => <button key={d} onClick={() => setSilenceDuration(d)} aria-pressed={silenceDuration === d} className={`rounded-lg border px-3 py-1.5 text-sm ${silenceDuration === d ? "border-red-500 bg-red-50 dark:bg-red-950/30" : "border-gray-300 dark:border-gray-700"}`}>{d}m</button>)}</div></div>
            </div>
            <div className="mt-4 flex justify-end gap-2"><button onClick={() => setShowSilenceForm(false)} className="rounded-lg border border-gray-300 px-4 py-2 text-sm dark:border-gray-700">{t("common.cancel")}</button><button onClick={() => setShowSilenceForm(false)} className="rounded-lg bg-red-600 px-4 py-2 text-sm font-medium text-white hover:bg-red-700">{t("alerts.create")}</button></div>
          </div>
        </div>
      )}
    </div>
  );
}
