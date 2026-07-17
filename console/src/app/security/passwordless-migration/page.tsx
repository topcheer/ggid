"use client";

import { useState, useCallback, useEffect } from "react";
import {
  KeyRound, Fingerprint, Shield, Loader2, AlertCircle, X, RefreshCw,
  TrendingUp, Users, CheckCircle, XCircle, Clock, Target, Gauge,
  ArrowRight, ChevronRight, Calendar, AlertTriangle, Zap,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";
import { authHeader } from "@/lib/auth-helpers";

const TENANT_ID = "00000000-0000-0000-0000-000000000001";

interface MigrationStats {
  total_users: number;
  password_only: number;
  mfa_enabled: number;
  passkey_enabled: number;
  both_mfa_passkey: number;
  passkey_coverage_pct: number;
  mfa_coverage_pct: number;
  readiness_score: number;
  risk_level: "low" | "medium" | "high";
}

interface MigrationFunnel {
  invited: number;
  registered: number;
  activated: number;
  success_rate_pct: number;
  drop_off_invited_registered: number;
  drop_off_registered_activated: number;
}

interface PromptConfig {
  prompt_after_login: boolean;
  prompt_frequency: "always" | "daily" | "weekly" | "once";
  force_deadline: string | null;
  skip_allowed: boolean;
  max_skips: number;
}

interface DisablePlan {
  id: string;
  scope: "tenant" | "group";
  target: string;
  disable_date: string;
  affected_users: number;
  status: "draft" | "scheduled" | "executed" | "cancelled";
  grace_period_days: number;
}

const riskConfig = {
  low: { color: "text-green-600", bg: "bg-green-100 dark:bg-green-900/30", label: "Low Risk — Ready" },
  medium: { color: "text-yellow-600", bg: "bg-yellow-100 dark:bg-yellow-900/30", label: "Medium Risk — Caution" },
  high: { color: "text-red-600", bg: "bg-red-100 dark:bg-red-900/30", label: "High Risk — Not Ready" },
};

type Tab = "overview" | "funnel" | "prompt" | "disable" | "readiness";

export default function PasswordlessMigrationPage() {
  const t = useTranslations();
  const [tab, setTab] = useState<Tab>("overview");
  const [stats, setStats] = useState<MigrationStats | null>(null);
  const [funnel, setFunnel] = useState<MigrationFunnel | null>(null);
  const [promptCfg, setPromptCfg] = useState<PromptConfig>({ prompt_after_login: true, prompt_frequency: "daily", force_deadline: null, skip_allowed: true, max_skips: 3 });
  const [plans, setPlans] = useState<DisablePlan[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  // Disable plan form
  const [showPlanForm, setShowPlanForm] = useState(false);
  const [planScope, setPlanScope] = useState<"tenant" | "group">("tenant");
  const [planTarget, setPlanTarget] = useState("");
  const [planDate, setPlanDate] = useState("");
  const [planGrace, setPlanGrace] = useState(14);

  const loadData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const h = { ...authHeader(), "X-Tenant-ID": TENANT_ID };
      const [statsRes, funnelRes, plansRes] = await Promise.all([
        fetch("/api/v1/auth/passwordless/stats", { headers: h }).catch(() => null),
        fetch("/api/v1/auth/passwordless/funnel", { headers: h }).catch(() => null),
        fetch("/api/v1/auth/passwordless/disable-plans", { headers: h }).catch(() => null),
      ]);
      if (statsRes?.ok) setStats(await statsRes.json());
      if (funnelRes?.ok) setFunnel(await funnelRes.json());
      if (plansRes?.ok) { const d = await plansRes.json(); setPlans(d.plans || d.items || []); }
    } catch { setError("Failed to load migration data"); }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { loadData(); }, [loadData]);

  const savePromptConfig = async () => {
    setSaving(true);
    try {
      await fetch("/api/v1/auth/passwordless/prompt-config", {
        method: "PUT",
        headers: { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": TENANT_ID },
        body: JSON.stringify(promptCfg),
      });
    } catch { /* noop */ }
    finally { setSaving(false); }
  };

  const createPlan = async () => {
    try {
      await fetch("/api/v1/auth/passwordless/disable-plans", {
        method: "POST",
        headers: { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": TENANT_ID },
        body: JSON.stringify({ scope: planScope, target: planTarget || "all", disable_date: planDate, grace_period_days: planGrace }),
      });
      setShowPlanForm(false); setPlanDate(""); loadData();
    } catch { setError("Failed to create plan"); }
  };

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><Fingerprint className="h-6 w-6 text-indigo-500" /> {t("passwordlessMigration.title")}</h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{t("passwordlessMigration.subtitle")}</p>
        </div>
        <button onClick={loadData} disabled={loading} aria-label="Refresh" className="flex items-center gap-2 rounded-lg border border-gray-300 px-3 py-2 text-sm text-gray-600 hover:bg-gray-50 dark:border-gray-700 dark:text-gray-300 dark:hover:bg-gray-800"><RefreshCw className={"h-4 w-4 " + (loading ? "animate-spin" : "")} /> Refresh</button>
      </div>

      {error && <div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400"><AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} aria-label="Dismiss" className="ml-auto"><X className="h-4 w-4" /></button></div>}

      <div className="flex gap-1 border-b border-gray-200 dark:border-gray-700 overflow-x-auto">
        {([
          { id: "overview" as Tab, label: "Overview", icon: Users },
          { id: "funnel" as Tab, label: "Migration Funnel", icon: TrendingUp },
          { id: "prompt" as Tab, label: "Prompt Strategy", icon: Zap },
          { id: "disable" as Tab, label: "Password Disable", icon: KeyRound },
          { id: "readiness" as Tab, label: "Readiness Score", icon: Gauge },
        ]).map(tb => { const Icon = tb.icon; return (
          <button key={tb.id} onClick={() => setTab(tb.id)} aria-pressed={tab === tb.id} className={"flex items-center gap-1.5 border-b-2 px-4 py-2 text-sm font-medium transition whitespace-nowrap " + (tab === tb.id ? "border-indigo-600 text-indigo-600 dark:text-indigo-400" : "border-transparent text-gray-500 hover:text-gray-700 dark:hover:text-gray-300")}><Icon className="h-4 w-4" /> {tb.label}</button>
        ); })}
      </div>

      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-indigo-500" /></div> : (<>

      {/* OVERVIEW */}
      {tab === "overview" && stats && (
        <div className="space-y-4">
          {/* Auth method distribution */}
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
            <div className={cardCls + " flex items-center gap-4"}><div className="flex h-12 w-12 items-center justify-center rounded-xl bg-red-100 dark:bg-red-900/30"><KeyRound className="h-6 w-6 text-red-500" /></div><div><p className="text-xs font-semibold uppercase text-gray-400">Password Only</p><p className="text-2xl font-bold text-red-600">{stats.password_only}</p></div></div>
            <div className={cardCls + " flex items-center gap-4"}><div className="flex h-12 w-12 items-center justify-center rounded-xl bg-yellow-100 dark:bg-yellow-900/30"><Shield className="h-6 w-6 text-yellow-500" /></div><div><p className="text-xs font-semibold uppercase text-gray-400">MFA Enabled</p><p className="text-2xl font-bold text-yellow-600">{stats.mfa_enabled}</p></div></div>
            <div className={cardCls + " flex items-center gap-4"}><div className="flex h-12 w-12 items-center justify-center rounded-xl bg-green-100 dark:bg-green-900/30"><Fingerprint className="h-6 w-6 text-green-500" /></div><div><p className="text-xs font-semibold uppercase text-gray-400">Passkey</p><p className="text-2xl font-bold text-green-600">{stats.passkey_enabled}</p></div></div>
            <div className={cardCls + " flex items-center gap-4"}><div className="flex h-12 w-12 items-center justify-center rounded-xl bg-indigo-100 dark:bg-indigo-900/30"><Users className="h-6 w-6 text-indigo-500" /></div><div><p className="text-xs font-semibold uppercase text-gray-400">Total Users</p><p className="text-2xl font-bold">{stats.total_users}</p></div></div>
          </div>
          {/* Stacked bar */}
          <div className={cardCls}>
            <h2 className="mb-4 text-sm font-semibold uppercase text-gray-400">Authentication Method Distribution</h2>
            <div className="flex h-8 overflow-hidden rounded-lg">
              {stats.total_users > 0 && <>
                <div className="bg-red-500 flex items-center justify-center text-xs text-white font-medium" style={{ width: `${(stats.password_only / stats.total_users) * 100}%` }}>{((stats.password_only / stats.total_users) * 100).toFixed(0)}%</div>
                <div className="bg-yellow-500 flex items-center justify-center text-xs text-white font-medium" style={{ width: `${(stats.mfa_enabled / stats.total_users) * 100}%` }}>{((stats.mfa_enabled / stats.total_users) * 100).toFixed(0)}%</div>
                <div className="bg-green-500 flex items-center justify-center text-xs text-white font-medium" style={{ width: `${(stats.passkey_enabled / stats.total_users) * 100}%` }}>{((stats.passkey_enabled / stats.total_users) * 100).toFixed(0)}%</div>
              </>}
            </div>
            <div className="mt-2 flex gap-4 text-xs">
              <span className="flex items-center gap-1"><div className="h-3 w-3 rounded bg-red-500" /> Password</span>
              <span className="flex items-center gap-1"><div className="h-3 w-3 rounded bg-yellow-500" /> MFA</span>
              <span className="flex items-center gap-1"><div className="h-3 w-3 rounded bg-green-500" /> Passkey</span>
            </div>
          </div>
        </div>
      )}

      {/* FUNNEL */}
      {tab === "funnel" && funnel && (
        <div className="space-y-4">
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-4">
            {[
              { label: "Invited", value: funnel.invited, color: "text-blue-600", pct: 100 },
              { label: "Registered", value: funnel.registered, color: "text-indigo-600", pct: funnel.invited ? (funnel.registered / funnel.invited) * 100 : 0 },
              { label: "Activated", value: funnel.activated, color: "text-green-600", pct: funnel.invited ? (funnel.activated / funnel.invited) * 100 : 0 },
              { label: "Success Rate", value: `${funnel.success_rate_pct}%`, color: "text-purple-600", pct: funnel.success_rate_pct },
            ].map(s => (
              <div key={s.label} className={cardCls + " text-center"}>
                <p className="text-xs font-semibold uppercase text-gray-400">{s.label}</p>
                <p className={"mt-2 text-2xl font-bold " + s.color}>{s.value}</p>
                <div className="mt-2 h-1.5 overflow-hidden rounded-full bg-gray-200 dark:bg-gray-700"><div className="h-full rounded-full bg-indigo-500" style={{ width: `${s.pct}%` }} /></div>
              </div>
            ))}
          </div>
          {/* Visual funnel */}
          <div className={cardCls}>
            <h2 className="mb-4 text-sm font-semibold uppercase text-gray-400">Passkey Adoption Funnel</h2>
            <div className="space-y-2">
              {[
                { label: "Users Invited", value: funnel.invited, width: 100, color: "bg-blue-500" },
                { label: "Passkey Registered", value: funnel.registered, width: funnel.invited ? (funnel.registered / funnel.invited) * 100 : 0, color: "bg-indigo-500" },
                { label: "First Successful Login", value: funnel.activated, width: funnel.invited ? (funnel.activated / funnel.invited) * 100 : 0, color: "bg-green-500" },
              ].map((f, i) => (
                <div key={i} className="flex items-center gap-3">
                  <span className="w-40 text-xs text-gray-500 text-right">{f.label}</span>
                  <div className="flex-1"><div className={"h-8 rounded-lg flex items-center justify-end px-3 text-xs text-white font-medium transition-all " + f.color} style={{ width: `${Math.max(f.width, 10)}%` }}>{f.value}</div></div>
                </div>
              ))}
            </div>
            {(funnel.drop_off_invited_registered > 0 || funnel.drop_off_registered_activated > 0) && (
              <div className="mt-4 rounded-lg bg-yellow-50 p-3 dark:bg-yellow-950/20"><p className="flex items-center gap-1 text-xs text-yellow-700 dark:text-yellow-400"><AlertTriangle className="h-3 w-3" /> Drop-off: {funnel.drop_off_invited_registered} didn't register after invite, {funnel.drop_off_registered_activated} registered but never activated.</p></div>
            )}
          </div>
        </div>
      )}

      {/* PROMPT STRATEGY */}
      {tab === "prompt" && (
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
          <div className={cardCls}>
            <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Zap className="h-4 w-4" /> Passkey Prompt Configuration</h2>
            <div className="space-y-4">
              <label className="flex items-center justify-between"><span className="text-sm font-medium">Prompt after login</span><input type="checkbox" checked={promptCfg.prompt_after_login} onChange={e => setPromptCfg({ ...promptCfg, prompt_after_login: e.target.checked })} className="h-5 w-5 rounded" /></label>
              <div><label className="text-sm font-medium">Prompt Frequency</label><select aria-label="Prompt frequency" value={promptCfg.prompt_frequency} onChange={e => setPromptCfg({ ...promptCfg, prompt_frequency: e.target.value as PromptConfig["prompt_frequency"] })} className="mt-1 block w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm"><option value="always">Every login</option><option value="daily">Once per day</option><option value="weekly">Once per week</option><option value="once">Only once</option></select></div>
              <label className="flex items-center justify-between"><span className="text-sm font-medium">Allow user to skip</span><input type="checkbox" checked={promptCfg.skip_allowed} onChange={e => setPromptCfg({ ...promptCfg, skip_allowed: e.target.checked })} className="h-5 w-5 rounded" /></label>
              {promptCfg.skip_allowed && <div><label className="text-sm font-medium">Max skips before forced</label><input aria-label="Max skips" type="number" min={0} max={20} value={promptCfg.max_skips} onChange={e => setPromptCfg({ ...promptCfg, max_skips: parseInt(e.target.value) || 0 })} className="mt-1 w-24 rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm" /></div>}
              <div><label className="text-sm font-medium">Force deadline (optional)</label><input aria-label="Force deadline" type="date" value={promptCfg.force_deadline || ""} onChange={e => setPromptCfg({ ...promptCfg, force_deadline: e.target.value || null })} className="mt-1 rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm" /></div>
              <button onClick={savePromptConfig} disabled={saving} className="flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50">{saving ? <Loader2 className="h-4 w-4 animate-spin" /> : <CheckCircle className="h-4 w-4" />} Save Strategy</button>
            </div>
          </div>
          <div className={cardCls}>
            <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Target className="h-4 w-4" /> Strategy Preview</h2>
            <div className="space-y-3">
              <div className="rounded-lg bg-gray-50 p-3 dark:bg-gray-900/50">
                <p className="text-sm text-gray-700 dark:text-gray-300">When a password user logs in:</p>
                <ol className="mt-2 space-y-1 text-xs text-gray-500">
                  <li className="flex items-center gap-2"><ChevronRight className="h-3 w-3" /> {promptCfg.prompt_after_login ? "Show passkey upgrade prompt" : "No prompt shown"}</li>
                  {promptCfg.prompt_after_login && <><li className="flex items-center gap-2"><ChevronRight className="h-3 w-3" /> Frequency: {promptCfg.prompt_frequency}</li>
                  {promptCfg.skip_allowed && <li className="flex items-center gap-2"><ChevronRight className="h-3 w-3" /> User can skip (max {promptCfg.max_skips} times)</li>}
                  {promptCfg.force_deadline && <li className="flex items-center gap-2 text-red-500"><AlertTriangle className="h-3 w-3" /> Force upgrade by {new Date(promptCfg.force_deadline).toLocaleDateString()}</li>}</>}
                </ol>
              </div>
              <div className="rounded-lg bg-blue-50 p-3 dark:bg-blue-950/30"><p className="text-xs text-blue-700 dark:text-blue-400"><Fingerprint className="inline h-3 w-3 mr-1" /> Conditional Create is active — browser auto-suggests passkey creation after password login (FIDO L3).</p></div>
            </div>
          </div>
        </div>
      )}

      {/* PASSWORD DISABLE */}
      {tab === "disable" && (
        <>
          <div className="flex justify-end"><button onClick={() => setShowPlanForm(true)} className="flex items-center gap-2 rounded-lg bg-red-600 px-3 py-2 text-sm font-medium text-white hover:bg-red-700"><KeyRound className="h-4 w-4" /> New Disable Plan</button></div>
          <div className={cardCls}>
            <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Calendar className="h-4 w-4" /> Password Disable Plans</h2>
            {plans.length === 0 ? <div className="py-8 text-center"><KeyRound className="mx-auto h-10 w-10 text-gray-300" /><p className="mt-3 text-sm text-gray-400">No disable plans scheduled.</p></div> : (
              <div className="space-y-3">{plans.map(p => (
                <div key={p.id} className="flex items-center justify-between rounded-lg border p-4 dark:border-gray-700">
                  <div className="flex items-center gap-3"><div className={"h-10 w-10 rounded-lg flex items-center justify-center " + (p.status === "executed" ? "bg-gray-100 dark:bg-gray-700" : "bg-red-100 dark:bg-red-900/30")}><KeyRound className={"h-5 w-5 " + (p.status === "executed" ? "text-gray-400" : "text-red-500")} /></div><div><div className="flex items-center gap-2"><span className="font-medium text-sm">{p.scope === "tenant" ? "Entire tenant" : `Group: ${p.target}`}</span><span className={"px-2 py-0.5 rounded text-xs " + (p.status === "scheduled" ? "bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-400" : "bg-gray-100 dark:bg-gray-800 text-gray-400")}>{p.status}</span></div><p className="text-xs text-gray-400">Disable date: {new Date(p.disable_date).toLocaleDateString()} · Grace: {p.grace_period_days}d · {p.affected_users} users affected</p></div></div>
                  {p.status === "scheduled" && <span className="text-xs text-red-500 flex items-center gap-1"><Clock className="h-3 w-3" /> {Math.ceil((new Date(p.disable_date).getTime() - Date.now()) / 86400000)}d left</span>}
                </div>
              ))}</div>
            )}
          </div>
        </>
      )}

      {/* READINESS */}
      {tab === "readiness" && stats && (
        <div className="space-y-4">
          <div className={cardCls + " text-center"}>
            <h2 className="mb-4 text-sm font-semibold uppercase text-gray-400">Passwordless Readiness Score</h2>
            <div className="mx-auto w-48 h-48 relative">
              <svg width="192" height="192" viewBox="0 0 192 192" className="-rotate-90">
                <circle cx="96" cy="96" r="80" fill="none" stroke="#e5e7eb" strokeWidth="12" className="dark:stroke-gray-700" />
                <circle cx="96" cy="96" r="80" fill="none" stroke={stats.readiness_score >= 80 ? "#16a34a" : stats.readiness_score >= 50 ? "#eab308" : "#dc2626"} strokeWidth="12" strokeLinecap="round" strokeDasharray={`${(stats.readiness_score / 100) * 502} 502`} />
              </svg>
              <div className="absolute inset-0 flex flex-col items-center justify-center"><span className={"text-4xl font-bold " + (stats.readiness_score >= 80 ? "text-green-600" : stats.readiness_score >= 50 ? "text-yellow-600" : "text-red-600")}>{stats.readiness_score}</span><span className="text-xs text-gray-400">/ 100</span></div>
            </div>
            <div className="mt-4"><span className={"inline-flex items-center gap-1.5 rounded-full px-4 py-2 text-sm font-bold " + riskConfig[stats.risk_level].bg + " " + riskConfig[stats.risk_level].color}><Shield className="h-4 w-4" /> {riskConfig[stats.risk_level].label}</span></div>
          </div>
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <div className={cardCls}><div className="flex items-center justify-between"><span className="text-sm font-medium">Passkey Coverage</span><span className="text-lg font-bold text-green-600">{stats.passkey_coverage_pct}%</span></div><div className="mt-2 h-3 overflow-hidden rounded-full bg-gray-200 dark:bg-gray-700"><div className="h-full rounded-full bg-green-500" style={{ width: `${stats.passkey_coverage_pct}%` }} /></div></div>
            <div className={cardCls}><div className="flex items-center justify-between"><span className="text-sm font-medium">MFA Coverage</span><span className="text-lg font-bold text-yellow-600">{stats.mfa_coverage_pct}%</span></div><div className="mt-2 h-3 overflow-hidden rounded-full bg-gray-200 dark:bg-gray-700"><div className="h-full rounded-full bg-yellow-500" style={{ width: `${stats.mfa_coverage_pct}%` }} /></div></div>
          </div>
          {stats.risk_level !== "low" && (
            <div className="rounded-xl border border-yellow-300 bg-yellow-50 p-4 dark:border-yellow-700 dark:bg-yellow-950/30"><div className="flex items-start gap-3"><AlertTriangle className="h-5 w-5 shrink-0 text-yellow-600" /><div><p className="text-sm font-semibold text-yellow-800 dark:text-yellow-400">Not Ready to Disable Passwords</p><p className="mt-1 text-xs text-yellow-700 dark:text-yellow-500">Increase passkey coverage to 80%+ and MFA coverage to 95%+ before scheduling password disable. Current passkey-only users will be locked out.</p></div></div></div>
          )}
        </div>
      )}

      {tab === "overview" && !stats && <div className={cardCls}><div className="py-12 text-center"><Users className="mx-auto h-12 w-12 text-gray-300" /><p className="mt-4 text-sm text-gray-400">No migration data. Backend endpoint pending.</p></div></div>}
      {tab === "funnel" && !funnel && <div className={cardCls}><div className="py-12 text-center"><TrendingUp className="mx-auto h-12 w-12 text-gray-300" /><p className="mt-4 text-sm text-gray-400">No funnel data available.</p></div></div>}
      {tab === "readiness" && !stats && <div className={cardCls}><div className="py-12 text-center"><Gauge className="mx-auto h-12 w-12 text-gray-300" /><p className="mt-4 text-sm text-gray-400">No readiness data.</p></div></div>}

      </>)}

      {/* Disable plan dialog */}
      {showPlanForm && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setShowPlanForm(false)}>
          <div role="dialog" aria-modal="true" className="mx-4 w-full max-w-md rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={e => e.stopPropagation()}>
            <h3 className="flex items-center gap-2 text-lg font-semibold text-gray-900 dark:text-white"><KeyRound className="h-5 w-5 text-red-500" /> Password Disable Plan</h3>
            <div className="mt-3 rounded-lg bg-red-50 p-3 dark:bg-red-950/30"><p className="text-xs text-red-700 dark:text-red-400"><AlertTriangle className="inline h-3 w-3 mr-1" /> Passwords will be disabled for affected users. They must have a passkey or alternative login method.</p></div>
            <div className="mt-4 space-y-3">
              <div><label className="text-sm font-medium">Scope</label><select aria-label="Plan scope" value={planScope} onChange={e => setPlanScope(e.target.value as "tenant" | "group")} className="mt-1 block w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm"><option value="tenant">Entire Tenant</option><option value="group">Specific Group</option></select></div>
              {planScope === "group" && <div><label className="text-sm font-medium">Group Name</label><input aria-label="Group target" type="text" value={planTarget} onChange={e => setPlanTarget(e.target.value)} placeholder="engineering" className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm" /></div>}
              <div><label className="text-sm font-medium">Disable Date *</label><input aria-label="Disable date" type="date" value={planDate} onChange={e => setPlanDate(e.target.value)} className="mt-1 rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm" /></div>
              <div><label className="text-sm font-medium">Grace Period (days)</label><input aria-label="Grace period" type="number" min={0} max={90} value={planGrace} onChange={e => setPlanGrace(parseInt(e.target.value) || 0)} className="mt-1 w-24 rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm" /></div>
            </div>
            <div className="mt-4 flex justify-end gap-2"><button onClick={() => setShowPlanForm(false)} className="rounded-lg border border-gray-300 px-4 py-2 text-sm dark:border-gray-700">Cancel</button><button onClick={createPlan} disabled={!planDate} className="rounded-lg bg-red-600 px-4 py-2 text-sm font-medium text-white hover:bg-red-700 disabled:opacity-50">Schedule Disable</button></div>
          </div>
        </div>
      )}
    </div>
  );
}
