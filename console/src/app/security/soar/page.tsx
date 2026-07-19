"use client";
import { useState, useCallback, useEffect, useRef } from "react";
import {
  Zap, Loader2, AlertCircle, X, RefreshCw, Plus, Trash2, Check,
  Shield, Activity, ChevronRight, Clock, Play, Ban, Lock,
  CheckCircle2, XCircle, AlertTriangle, Bell, UserX, Globe,
  Rocket, FileText, Copy,
} from "lucide-react";
import { authHeader } from "@/lib/auth-helpers";
import { useTranslations } from "@/lib/i18n";

const TENANT_ID = "00000000-0000-0000-0000-000000000001";

interface PlaybookStep { order: number; action: string; target: string; delay_seconds?: number; }
interface Playbook { id: string; name: string; trigger: string; steps: PlaybookStep[]; enabled: boolean; created_at: string; }

type Tab = "playbooks" | "executions" | "catalog" | "templates";

const ACTION_ICONS: Record<string, typeof Ban> = {
  revoke_session: Ban, revoke_sessions: Ban, lock_account: Lock, disable_account: Lock,
  step_up_mfa: Shield, notify_soc: Bell, create_incident: AlertTriangle, block_ip: Globe, isolate_user: UserX,
};
const ACTION_COLORS: Record<string, string> = {
  revoke_session: "text-red-500", revoke_sessions: "text-red-500", lock_account: "text-red-500", disable_account: "text-red-500",
  step_up_mfa: "text-blue-500", notify_soc: "text-yellow-500", create_incident: "text-orange-500", block_ip: "text-red-500", isolate_user: "text-red-500",
};

const ACTION_CATALOG = [
  { action: "revoke_session", icon: Ban, color: "text-red-500", bg: "bg-red-100 dark:bg-red-900/30", desc: "Terminate active user sessions immediately", example: "Use when session token is compromised" },
  { action: "lock_account", icon: Lock, color: "text-red-500", bg: "bg-red-100 dark:bg-red-900/30", desc: "Disable user account login", example: "Use for confirmed account takeover" },
  { action: "step_up_mfa", icon: Shield, color: "text-blue-500", bg: "bg-blue-100 dark:bg-blue-900/30", desc: "Require additional MFA verification", example: "Use for suspicious but not confirmed activity" },
  { action: "notify_soc", icon: Bell, color: "text-yellow-500", bg: "bg-yellow-100 dark:bg-yellow-900/30", desc: "Send alert to Security Operations Center", example: "Use for events requiring human review" },
  { action: "create_incident", icon: AlertTriangle, color: "text-orange-500", bg: "bg-orange-100 dark:bg-orange-900/30", desc: "Open a security incident ticket", example: "Use for events needing investigation" },
  { action: "block_ip", icon: Globe, color: "text-red-500", bg: "bg-red-100 dark:bg-red-900/30", desc: "Add IP to denylist at gateway", example: "Use for known malicious source IPs" },
];

const TEMPLATES = [
  { name: "MFA Fatigue Response", trigger: "mfa_fatigue_detected", steps: [{ order: 1, action: "lock_account", target: "user" }, { order: 2, action: "notify_soc", target: "soc" }], desc: "When MFA fatigue attack detected, lock account and alert SOC" },
  { name: "Impossible Travel", trigger: "impossible_travel", steps: [{ order: 1, action: "step_up_mfa", target: "user" }, { order: 2, action: "block_ip", target: "source_ip" }], desc: "On impossible travel detection, require step-up MFA and block source IP" },
  { name: "Mass Data Export", trigger: "bulk_export_detected", steps: [{ order: 1, action: "revoke_session", target: "session" }, { order: 2, action: "create_incident", target: "incident" }], desc: "When bulk data export detected, revoke session and create incident" },
  { name: "Credential Stuffing", trigger: "credential_stuffing", steps: [{ order: 1, action: "lock_account", target: "user" }, { order: 2, action: "block_ip", target: "source_ip" }, { order: 3, action: "notify_soc", target: "soc" }], desc: "On credential stuffing burst, lock account, block IP, notify SOC" },
];

export default function SOARPage() {
  const t = useTranslations();
  const [tab, setTab] = useState<Tab>("playbooks");
  const [playbooks, setPlaybooks] = useState<Playbook[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [actionLoading, setActionLoading] = useState<string | null>(null);
  const [autoRefresh, setAutoRefresh] = useState(true);
  const refreshTimer = useRef<ReturnType<typeof setInterval> | null>(null);

  // Create form
  const [showForm, setShowForm] = useState(false);
  const [fName, setFName] = useState("");
  const [fTrigger, setFTrigger] = useState("critical");
  const [fActions, setFActions] = useState<string[]>([]);

  // Executions (demo data)
  const [executions, setExecutions] = useState([] as any[]);

  const H = { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": TENANT_ID };
  const h = { ...authHeader(), "X-Tenant-ID": TENANT_ID };
  const card = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  const loadData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/v1/audit/itdr/playbooks", { headers: h }).catch(() => null);
      if (res?.ok) { const d = await res.json(); setPlaybooks(d.playbooks || []); }
    } catch { setError(t("soar.loadError")); }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { loadData(); }, [loadData]);

  useEffect(() => {
    if (autoRefresh && tab === "executions") {
      refreshTimer.current = setInterval(() => loadData(), 10000);
      return () => { if (refreshTimer.current) clearInterval(refreshTimer.current); };
    }
  }, [autoRefresh, tab, loadData]);

  const createPlaybook = async () => {
    if (!fName) return;
    setActionLoading("create");
    try {
      await fetch("/api/v1/audit/itdr/playbooks", { method: "POST", headers: H, body: JSON.stringify({ name: fName, trigger: fTrigger, steps: fActions.map((a: any, i: number) => ({ order: i + 1, action: a, target: "auto" })), enabled: true }) });
      setShowForm(false); setFName(""); setFTrigger("critical"); setFActions([]);
      loadData();
    } catch { setError(t("soar.createError")); }
    finally { setActionLoading(null); }
  };

  const deletePlaybook = async (id: string) => {
    setActionLoading(`del-${id}`);
    try { await fetch(`/api/v1/audit/itdr/playbooks/${id}`, { method: "DELETE", headers: h }); loadData(); }
    catch { /* noop */ }
    finally { setActionLoading(null); }
  };

  const deployTemplate = async (tmpl: typeof TEMPLATES[0]) => {
    setActionLoading(`tmpl-${tmpl.name}`);
    try {
      await fetch("/api/v1/audit/itdr/playbooks", { method: "POST", headers: H, body: JSON.stringify({ name: tmpl.name, trigger: tmpl.trigger, steps: tmpl.steps, enabled: true }) });
      loadData(); setTab("playbooks");
    } catch { setError(t("soar.deployError")); }
    finally { setActionLoading(null); }
  };

  const toggleAction = (action: string) => setFActions(prev => prev.includes(action) ? prev.filter(a => a !== action) : [...prev, action]);
  const activePb = playbooks.filter(p => p.enabled);

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white">
            <Rocket className="h-6 w-6 text-pink-500" /> {t("soar.title")}
          </h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{t("soar.subtitle")}</p>
        </div>
        <div className="flex items-center gap-2">
          {tab === "executions" && <button onClick={() => setAutoRefresh(!autoRefresh)} aria-pressed={autoRefresh} className={`flex items-center gap-1 rounded-lg px-2 py-1 text-xs font-medium ${autoRefresh ? "bg-green-100 text-green-700 dark:bg-green-900/30" : "bg-gray-100 dark:bg-gray-800 text-gray-400"}`}><Activity className="h-3 w-3" /> {autoRefresh ? "Live" : "Paused"}</button>}
          <button onClick={() => setShowForm(true)} className="flex items-center gap-1 rounded-lg bg-pink-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-pink-700"><Plus className="h-3 w-3" /> {t("soar.newPlaybook")}</button>
        </div>
      </div>

      {error && (
        <div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400">
          <AlertCircle className="h-4 w-4 shrink-0" />{error}
          <button onClick={() => setError(null)} aria-label="Dismiss" className="ml-auto"><X className="h-4 w-4" /></button>
        </div>
      )}

      <div className="flex gap-1 border-b border-gray-200 dark:border-gray-700 overflow-x-auto">
        {([
          { id: "playbooks" as Tab, label: `${t("soar.playbooks")} (${activePb.length})`, icon: Zap },
          { id: "executions" as Tab, label: t("soar.executions"), icon: Activity },
          { id: "catalog" as Tab, label: t("soar.catalog"), icon: Shield },
          { id: "templates" as Tab, label: t("soar.templates"), icon: FileText },
        ]).map(tb => {
          const Icon = tb.icon;
          return (
            <button key={tb.id} onClick={() => setTab(tb.id)} aria-pressed={tab === tb.id}
              className={`flex items-center gap-1.5 border-b-2 px-4 py-2 text-sm font-medium transition whitespace-nowrap ${tab === tb.id ? "border-pink-600 text-pink-600 dark:text-pink-400" : "border-transparent text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"}`}>
              <Icon className="h-4 w-4" /> {tb.label}
            </button>
          );
        })}
      </div>

      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-pink-500" /></div> : (<>

      {/* ════ PLAYBOOKS ════ */}
      {tab === "playbooks" && (
        <div>
          {playbooks.length === 0 ? (
            <div className={card}><div className="py-12 text-center"><Rocket className="mx-auto h-12 w-12 text-gray-300" /><p className="mt-4 text-sm text-gray-400">{t("soar.noPlaybooks")}</p><button onClick={() => setTab("templates")} className="mt-3 text-sm text-pink-600 hover:underline">{t("soar.browseTemplates")}</button></div></div>
          ) : (
            <div className="space-y-2">
              {playbooks.map(pb => (
                <div key={pb.id} className={`${card} flex items-center justify-between`}>
                  <div className="flex items-center gap-3 flex-1 min-w-0">
                    <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-pink-100 dark:bg-pink-900/30"><Rocket className="h-4 w-4 text-pink-500" /></div>
                    <div className="min-w-0">
                      <div className="flex items-center gap-2"><span className="font-medium text-sm">{pb.name}</span><span className={`px-1.5 py-0.5 rounded text-xs font-medium ${pb.enabled ? "bg-green-100 dark:bg-green-900/30 text-green-600" : "bg-gray-100 dark:bg-gray-800 text-gray-400"}`}>{pb.enabled ? "on" : "off"}</span></div>
                      <p className="text-xs text-gray-400">{t("soar.trigger")}: <code className="font-mono">{pb.trigger}</code> · {(pb.steps || []).length} {t("soar.steps")}</p>
                    </div>
                  </div>
                  <div className="flex items-center gap-1">
                    {(pb.steps || []).slice(0, 4).map((s: any, i: number) => { const AIcon = ACTION_ICONS[s.action] || Shield; return <span key={i} className="flex h-6 w-6 items-center justify-center rounded bg-gray-100 dark:bg-gray-700"><AIcon className={`h-3 w-3 ${ACTION_COLORS[s.action] || "text-gray-400"}`} /></span>; })}
                    <button onClick={() => deletePlaybook(pb.id)} disabled={actionLoading === `del-${pb.id}`} aria-label={"Delete " + pb.name} className="ml-2 rounded p-1 text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20">{actionLoading === `del-${pb.id}` ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Trash2 className="h-3.5 w-3.5" />}</button>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      {/* ════ EXECUTIONS ════ */}
      {tab === "executions" && (
        <div className="space-y-2">
          {executions.map(ex => {
            const cfg = ex.status === "success" ? { color: "text-green-600", bg: "bg-green-100 dark:bg-green-900/30", icon: CheckCircle2 } : ex.status === "failed" ? { color: "text-red-600", bg: "bg-red-100 dark:bg-red-900/30", icon: XCircle } : { color: "text-blue-600", bg: "bg-blue-100 dark:bg-blue-900/30", icon: Loader2 };
            const SIcon = cfg.icon;
            return (
              <div key={ex.id} className={`${card} flex items-center justify-between`}>
                <div className="flex items-center gap-3">
                  <div className={`flex h-9 w-9 items-center justify-center rounded-lg ${cfg.bg}`}><SIcon className={`h-4 w-4 ${cfg.color} ${ex.status === "running" ? "animate-spin" : ""}`} /></div>
                  <div>
                    <div className="flex items-center gap-2"><span className="font-medium text-sm">{ex.playbook}</span><span className={`px-1.5 py-0.5 rounded text-xs font-medium ${cfg.bg} ${cfg.color}`}>{ex.status}</span></div>
                    <p className="text-xs text-gray-400">{t("soar.trigger")}: <code className="font-mono">{ex.trigger}</code> · {ex.user} · {new Date(ex.time).toLocaleTimeString()}</p>
                  </div>
                </div>
                <div className="text-right">
                  <span className="text-xs font-mono text-gray-500">{ex.actions} {t("soar.actionsTaken")}</span>
                  {ex.duration_ms > 0 && <p className="text-xs text-gray-400">{ex.duration_ms}ms</p>}
                </div>
              </div>
            );
          })}
        </div>
      )}

      {/* ════ CATALOG ════ */}
      {tab === "catalog" && (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {ACTION_CATALOG.map(a => {
            const AIcon = a.icon;
            return (
              <div key={a.action} className={card + " hover:shadow-md transition"}>
                <div className="flex items-center gap-3 mb-2">
                  <div className={`flex h-10 w-10 items-center justify-center rounded-lg ${a.bg}`}><AIcon className={`h-5 w-5 ${a.color}`} /></div>
                  <div><h3 className="font-semibold text-sm font-mono">{a.action}</h3></div>
                </div>
                <p className="text-xs text-gray-500 dark:text-gray-400">{a.desc}</p>
                <div className="mt-2 rounded-lg bg-gray-50 dark:bg-gray-900/50 p-2 text-xs text-gray-400">{a.example}</div>
              </div>
            );
          })}
        </div>
      )}

      {/* ════ TEMPLATES ════ */}
      {tab === "templates" && (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          {TEMPLATES.map(tmpl => (
            <div key={tmpl.name} className={card + " hover:shadow-md transition"}>
              <div className="flex items-start justify-between">
                <div className="flex items-center gap-3">
                  <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-pink-100 dark:bg-pink-900/30"><Rocket className="h-5 w-5 text-pink-500" /></div>
                  <div><h3 className="font-semibold text-sm">{tmpl.name}</h3><p className="text-xs text-gray-400">{t("soar.trigger")}: <code className="font-mono">{tmpl.trigger}</code></p></div>
                </div>
                <button onClick={() => deployTemplate(tmpl)} disabled={actionLoading === `tmpl-${tmpl.name}`} className="flex items-center gap-1 rounded-lg bg-pink-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-pink-700 disabled:opacity-50">
                  {actionLoading === `tmpl-${tmpl.name}` ? <Loader2 className="h-3 w-3 animate-spin" /> : <Play className="h-3 w-3" />} {t("soar.deploy")}
                </button>
              </div>
              <p className="mt-2 text-xs text-gray-500 dark:text-gray-400">{tmpl.desc}</p>
              <div className="mt-3 flex items-center gap-1">
                {tmpl.steps.map((s: any, i: number) => { const AIcon = ACTION_ICONS[s.action] || Shield; return (<span key={i} className="flex items-center gap-1">{i > 0 && <ChevronRight className="h-3 w-3 text-gray-300" />}<span className={`flex items-center gap-1 rounded px-1.5 py-0.5 text-xs font-mono bg-gray-100 dark:bg-gray-700`}><AIcon className={`h-3 w-3 ${ACTION_COLORS[s.action] || "text-gray-400"}`} /> {s.action}</span></span>); })}
              </div>
            </div>
          ))}
        </div>
      )}

      </>)}

      {/* Create playbook modal */}
      {showForm && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setShowForm(false)}>
          <div role="dialog" aria-modal="true" className="mx-4 w-full max-w-md rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={e => e.stopPropagation()}>
            <h3 className="flex items-center gap-2 text-lg font-semibold text-gray-900 dark:text-white"><Plus className="h-5 w-5 text-pink-500" /> {t("soar.newPlaybook")}</h3>
            <div className="mt-4 space-y-3">
              <div><label className="text-sm font-medium">{t("soar.playbookName")}</label><input type="text" value={fName} onChange={e => setFName(e.target.value)} placeholder="MFA Fatigue Response" className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm" autoFocus /></div>
              <div><label className="text-sm font-medium">{t("soar.triggerRule")}</label><input type="text" value={fTrigger} onChange={e => setFTrigger(e.target.value)} placeholder="critical or rule_id" className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm font-mono" /></div>
              <div><label className="text-sm font-medium">{t("soar.selectActions")}</label>
                <div className="mt-1 space-y-1">
                  {ACTION_CATALOG.map(a => (
                    <label key={a.action} className="flex items-center gap-2 rounded p-1.5 hover:bg-gray-50 dark:hover:bg-gray-900/50 cursor-pointer">
                      <input type="checkbox" checked={fActions.includes(a.action)} onChange={() => toggleAction(a.action)} className="rounded" />
                      <code className="text-xs font-mono">{a.action}</code>
                    </label>
                  ))}
                </div>
              </div>
            </div>
            <div className="mt-4 flex justify-end gap-2">
              <button onClick={() => setShowForm(false)} className="rounded-lg border border-gray-300 px-4 py-2 text-sm dark:border-gray-700">{t("common.cancel")}</button>
              <button onClick={createPlaybook} disabled={!fName || actionLoading === "create"} className="flex items-center gap-1 rounded-lg bg-pink-600 px-4 py-2 text-sm font-medium text-white hover:bg-pink-700 disabled:opacity-50">{actionLoading === "create" ? <Loader2 className="h-4 w-4 animate-spin" /> : <Check className="h-4 w-4" />} {t("soar.create")}</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
