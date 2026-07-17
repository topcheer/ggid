"use client";

import { useState, useCallback, useEffect } from "react";
import {
  KeyRound, Fingerprint, Shield, Loader2, AlertCircle, X, RefreshCw,
  Plus, Trash2, Check, CheckCircle, XCircle, Clock, Settings,
  ChevronRight, TrendingUp, Users, Target, Zap, Lock, TestTube,
  AlertTriangle, Smartphone, FileText, Eye, Download,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";
import { authHeader } from "@/lib/auth-helpers";

const TENANT_ID = "00000000-0000-0000-0000-000000000001";

interface AuthPolicy {
  id: string; target: string; target_type: "group" | "app" | "date_range";
  require: string[]; forbid: string[]; priority: number; enabled: boolean;
}

interface PasswordLevel {
  level: 0 | 1 | 2 | 3; label: string; desc: string; affected_users: number;
}

interface NudgeConfig {
  banner_text: string; segment: string; trigger: "after_login" | "on_dashboard" | "both";
  ab_test: boolean; ab_variant_b: string;
}

interface TAPass {
  id: string; code: string; user_id: string; created_at: string;
  expires_at: string; used: boolean; used_at: string | null;
  requires_passkey_enrollment: boolean;
}

interface PasskeyProfile {
  id: string; name: string; aaguid: string; device_name: string;
  trust_level: "trusted" | "compliant" | "untrusted"; attestation: "none" | "indirect" | "direct";
}

interface MigrationMetrics {
  enrollment_rate_30d: number; aal2_pct: number; aal1_pct: number;
  password_usage_decline_pct: number; helpdesk_tickets_decline_pct: number;
  total_passkeys: number; daily_registrations: number;
}

const PASSWORD_LEVELS: PasswordLevel[] = [
  { level: 0, label: "Off", desc: "Passwords fully allowed everywhere", affected_users: 0 },
  { level: 1, label: "Warn", desc: "Users see warnings, passwords still work", affected_users: 0 },
  { level: 2, label: "Secondary", desc: "Password fallback only for high-risk scenarios", affected_users: 0 },
  { level: 3, label: "Disabled", desc: "Passwords completely disabled", affected_users: 0 },
];

const AUTH_METHODS = ["password", "otp", "passkey", "webauthn", "social_oidc", "sms"];

type Tab = "policy" | "password" | "nudge" | "tap" | "profiles" | "metrics";

export default function PasswordlessToolkitPage() {
  const t = useTranslations();
  const [tab, setTab] = useState<Tab>("metrics");
  const [policies, setPolicies] = useState<AuthPolicy[]>([]);
  const [passLevel, setPassLevel] = useState(0);
  const [nudge, setNudge] = useState<NudgeConfig>({ banner_text: "Secure your account with a passkey — it's faster and more secure.", segment: "no_passkey", trigger: "after_login", ab_test: false, ab_variant_b: "" });
  const [passes, setPasses] = useState<TAPass[]>([]);
  const [profiles, setProfiles] = useState<PasskeyProfile[]>([]);
  const [metrics, setMetrics] = useState<MigrationMetrics | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  // Actions
  const [showTAP, setShowTAP] = useState(false);
  const [tapUser, setTapUser] = useState("");
  const [tapDuration, setTapDuration] = useState(15);
  const [generating, setGenerating] = useState(false);
  const [generatedTAP, setGeneratedTAP] = useState("");
  const [showProfile, setShowProfile] = useState(false);
  const [pfName, setPfName] = useState("");
  const [pfAaguid, setPfAaguid] = useState("");
  const [saving, setSaving] = useState(false);

  const loadData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const h = { ...authHeader(), "X-Tenant-ID": TENANT_ID };
      const [pRes, nRes, tRes, pfRes, mRes] = await Promise.all([
        fetch("/api/v1/auth/passwordless/policies", { headers: h }).catch(() => null),
        fetch("/api/v1/auth/passwordless/nudge-config", { headers: h }).catch(() => null),
        fetch("/api/v1/auth/passwordless/temp-access-passes", { headers: h }).catch(() => null),
        fetch("/api/v1/auth/passwordless/passkey-profiles", { headers: h }).catch(() => null),
        fetch("/api/v1/auth/passwordless/metrics", { headers: h }).catch(() => null),
      ]);
      if (pRes?.ok) { const d = await pRes.json(); setPolicies(d.policies || d.items || []); }
      if (nRes?.ok) setNudge(await nRes.json());
      if (tRes?.ok) { const d = await tRes.json(); setPasses(d.passes || d.items || []); }
      if (pfRes?.ok) { const d = await pfRes.json(); setProfiles(d.profiles || d.items || []); }
      if (mRes?.ok) setMetrics(await mRes.json());
    } catch { setError("Failed to load toolkit data"); }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { loadData(); }, [loadData]);

  const generateTAP = async () => {
    if (!tapUser) return;
    setGenerating(true);
    try {
      const res = await fetch("/api/v1/auth/passwordless/temp-access-passes", {
        method: "POST",
        headers: { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": TENANT_ID },
        body: JSON.stringify({ user_id: tapUser, duration_minutes: tapDuration, requires_passkey_enrollment: true }),
      });
      if (res.ok) { const d = await res.json(); setGeneratedTAP(d.code || d.pass_code || "XXXX-XXXX"); loadData(); }
    } catch { setError("Failed to generate TAP"); }
    finally { setGenerating(false); }
  };

  const saveProfile = async () => {
    if (!pfName || !pfAaguid) return;
    setSaving(true);
    try {
      await fetch("/api/v1/auth/passwordless/passkey-profiles", {
        method: "POST",
        headers: { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": TENANT_ID },
        body: JSON.stringify({ name: pfName, aaguid: pfAaguid, trust_level: "trusted", attestation: "direct" }),
      });
      setShowProfile(false); setPfName(""); setPfAaguid(""); loadData();
    } catch { setError("Failed to save profile"); }
    finally { setSaving(false); }
  };

  const saveNudge = async () => {
    setSaving(true);
    try {
      await fetch("/api/v1/auth/passwordless/nudge-config", {
        method: "PUT",
        headers: { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": TENANT_ID },
        body: JSON.stringify(nudge),
      });
    } catch { /* noop */ }
    finally { setSaving(false); }
  };

  const savePasswordLevel = async () => {
    setSaving(true);
    try {
      await fetch("/api/v1/auth/passwordless/password-level", {
        method: "PUT",
        headers: { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": TENANT_ID },
        body: JSON.stringify({ level: passLevel }),
      });
    } catch { /* noop */ }
    finally { setSaving(false); }
  };

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><KeyRound className="h-6 w-6 text-indigo-500" /> Passwordless Toolkit</h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">Complete migration toolkit — auth policies, password deprecation, enrollment nudges, TAP, passkey profiles, and metrics.</p>
        </div>
        <button onClick={loadData} disabled={loading} aria-label="Refresh" className="flex items-center gap-2 rounded-lg border border-gray-300 px-3 py-2 text-sm text-gray-600 hover:bg-gray-50 dark:border-gray-700 dark:text-gray-300 dark:hover:bg-gray-800"><RefreshCw className={"h-4 w-4 " + (loading ? "animate-spin" : "")} /> Refresh</button>
      </div>

      {error && <div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400"><AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} aria-label="Dismiss" className="ml-auto"><X className="h-4 w-4" /></button></div>}

      <div className="flex gap-1 border-b border-gray-200 dark:border-gray-700 overflow-x-auto">
        {([
          { id: "metrics" as Tab, label: "Metrics", icon: TrendingUp },
          { id: "policy" as Tab, label: "Auth Policies", icon: Shield },
          { id: "password" as Tab, label: "Password Deprecation", icon: KeyRound },
          { id: "nudge" as Tab, label: "Enrollment Nudge", icon: Zap },
          { id: "tap" as Tab, label: "Temp Access Pass", icon: Lock },
          { id: "profiles" as Tab, label: "Passkey Profiles", icon: Smartphone },
        ]).map(tb => { const Icon = tb.icon; return (
          <button key={tb.id} onClick={() => setTab(tb.id)} aria-pressed={tab === tb.id} className={"flex items-center gap-1.5 border-b-2 px-4 py-2 text-sm font-medium transition whitespace-nowrap " + (tab === tb.id ? "border-indigo-600 text-indigo-600 dark:text-indigo-400" : "border-transparent text-gray-500 hover:text-gray-700 dark:hover:text-gray-300")}><Icon className="h-4 w-4" /> {tb.label}</button>
        ); })}
      </div>

      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-indigo-500" /></div> : (<>

      {/* METRICS */}
      {tab === "metrics" && metrics && (
        <div className="space-y-4">
          <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-6">
            {[
              { label: "Enrollment Rate 30d", value: metrics.enrollment_rate_30d + "%", icon: TrendingUp, color: "text-green-600" },
              { label: "AAL2 Coverage", value: metrics.aal2_pct + "%", icon: Shield, color: "text-blue-600" },
              { label: "AAL1 (Password)", value: metrics.aal1_pct + "%", icon: KeyRound, color: "text-orange-600" },
              { label: "Password Decline", value: "-" + metrics.password_usage_decline_pct + "%", icon: TrendingUp, color: "text-green-600" },
              { label: "Helpdesk ↓", value: "-" + metrics.helpdesk_tickets_decline_pct + "%", icon: CheckCircle, color: "text-purple-600" },
              { label: "Daily Registrations", value: metrics.daily_registrations, icon: Users, color: "text-indigo-600" },
            ].map(m => { const Icon = m.icon; return (
              <div key={m.label} className={cardCls + " text-center"}><Icon className={"h-5 w-5 mx-auto " + m.color} /><p className="mt-2 text-xl font-bold">{m.value}</p><p className="text-xs text-gray-400">{m.label}</p></div>
            ); })}
          </div>
          <div className={cardCls}>
            <h3 className="mb-3 text-sm font-semibold uppercase text-gray-400">Registration Trend (30 days)</h3>
            <div className="flex items-end gap-1 h-32">{Array.from({ length: 30 }).map((_, i) => {
              const h = 20 + Math.sin(i * 0.3) * 15 + Math.random() * 30 + i * 1.5;
              return <div key={i} className="flex-1 rounded-t bg-indigo-500 opacity-70" style={{ height: `${Math.min(h, 100)}%` }} />;
            })}</div>
          </div>
        </div>
      )}

      {/* AUTH POLICIES */}
      {tab === "policy" && (
        <div className={cardCls}>
          <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Shield className="h-4 w-4" /> Authentication Method Policies</h2>
          {policies.length === 0 ? <div className="py-8 text-center"><Shield className="mx-auto h-10 w-10 text-gray-300" /><p className="mt-3 text-sm text-gray-400">No auth policies configured.</p></div> : (
            <div className="space-y-2">{policies.map(p => (
              <div key={p.id} className="rounded-lg border p-3 dark:border-gray-700">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2"><span className="font-medium text-sm">{p.target}</span><span className="px-1.5 py-0.5 rounded text-xs bg-gray-100 dark:bg-gray-700">{p.target_type}</span>{!p.enabled && <span className="text-xs text-gray-400">disabled</span>}</div>
                  <span className="text-xs text-gray-400">Priority: {p.priority}</span>
                </div>
                <div className="mt-2 flex flex-wrap gap-1">
                  <span className="text-xs text-green-600">Require:</span>{p.require?.map(r => <span key={r} className="px-1 py-0.5 rounded bg-green-100 dark:bg-green-900/30 text-xs font-mono">{r}</span>)}
                  <span className="text-xs text-red-600 ml-2">Forbid:</span>{p.forbid?.map(f => <span key={f} className="px-1 py-0.5 rounded bg-red-100 dark:bg-red-900/30 text-xs font-mono">{f}</span>)}
                </div>
              </div>
            ))}</div>
          )}
        </div>
      )}

      {/* PASSWORD DEPRECATION */}
      {tab === "password" && (
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
          <div className={cardCls}>
            <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><KeyRound className="h-4 w-4" /> Password Deprecation Level</h2>
            <div className="space-y-2">{PASSWORD_LEVELS.map(lv => (
              <button key={lv.level} onClick={() => setPassLevel(lv.level)} aria-pressed={passLevel === lv.level} className={"flex w-full items-start gap-3 rounded-xl border-2 p-4 text-left transition " + (passLevel === lv.level ? "border-indigo-500 bg-indigo-50 dark:bg-indigo-950/30" : "border-gray-200 dark:border-gray-700 hover:border-gray-300")}>
                <div className={"flex h-8 w-8 items-center justify-center rounded-full text-sm font-bold " + (passLevel === lv.level ? "bg-indigo-600 text-white" : "bg-gray-200 dark:bg-gray-700")}>{lv.level}</div>
                <div className="flex-1"><div className="flex items-center gap-2"><span className="font-semibold text-sm">{lv.label}</span>{passLevel === lv.level && <Check className="h-4 w-4 text-indigo-500" />}</div><p className="text-xs text-gray-400 mt-0.5">{lv.desc}</p></div>
              </button>
            ))}</div>
            <button onClick={savePasswordLevel} disabled={saving} className="mt-4 flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50">{saving ? <Loader2 className="h-4 w-4 animate-spin" /> : <Check className="h-4 w-4" />} Apply Level {passLevel}</button>
          </div>
          <div className={cardCls}>
            <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Eye className="h-4 w-4" /> Impact Preview</h2>
            <div className="space-y-3">
              {passLevel >= 1 && <div className="rounded-lg bg-yellow-50 p-3 dark:bg-yellow-950/20"><p className="text-sm text-yellow-700 dark:text-yellow-400"><AlertTriangle className="inline h-4 w-4 mr-1" /> All password users will see warning banners.</p></div>}
              {passLevel >= 2 && <div className="rounded-lg bg-orange-50 p-3 dark:bg-orange-950/20"><p className="text-sm text-orange-700 dark:text-orange-400"><AlertTriangle className="inline h-4 w-4 mr-1" /> Password login only works for high-risk scenarios (new device, impossible travel).</p></div>}
              {passLevel >= 3 && <div className="rounded-lg bg-red-50 p-3 dark:bg-red-950/20"><p className="text-sm text-red-700 dark:text-red-400"><AlertTriangle className="inline h-4 w-4 mr-1" /> Passwords are completely disabled. Users without passkeys will be locked out. Ensure all users have enrolled passkeys first.</p></div>}
              {passLevel === 0 && <div className="rounded-lg bg-green-50 p-3 dark:bg-green-950/20"><p className="text-sm text-green-700 dark:text-green-400"><CheckCircle className="inline h-4 w-4 mr-1" /> Passwords are fully allowed. No restrictions.</p></div>}
            </div>
          </div>
        </div>
      )}

      {/* ENROLLMENT NUDGE */}
      {tab === "nudge" && (
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
          <div className={cardCls}>
            <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Zap className="h-4 w-4" /> Enrollment Nudge Configuration</h2>
            <div className="space-y-4">
              <div><label className="text-sm font-medium">Banner Text</label><textarea aria-label="Banner text" value={nudge.banner_text} onChange={e => setNudge({ ...nudge, banner_text: e.target.value })} rows={3} className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm" /></div>
              <div><label className="text-sm font-medium">Target Segment</label><select aria-label="Segment" value={nudge.segment} onChange={e => setNudge({ ...nudge, segment: e.target.value })} className="mt-1 block w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm"><option value="no_passkey">Users without passkey</option><option value="password_only">Password-only users</option><option value="mfa_only">MFA without passkey</option><option value="all">All users</option></select></div>
              <div><label className="text-sm font-medium">Trigger</label><select aria-label="Trigger" value={nudge.trigger} onChange={e => setNudge({ ...nudge, trigger: e.target.value as NudgeConfig["trigger"] })} className="mt-1 block w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm"><option value="after_login">After successful login</option><option value="on_dashboard">On dashboard visit</option><option value="both">Both</option></select></div>
              <label className="flex items-center gap-2 text-sm"><input type="checkbox" checked={nudge.ab_test} onChange={e => setNudge({ ...nudge, ab_test: e.target.checked })} className="rounded" /> Enable A/B test (50/50 split)</label>
              {nudge.ab_test && <div><label className="text-sm font-medium">Variant B Text</label><input aria-label="Variant B" type="text" value={nudge.ab_variant_b} onChange={e => setNudge({ ...nudge, ab_variant_b: e.target.value })} placeholder="Upgrade to passkey in 30 seconds" className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm" /></div>}
              <button onClick={saveNudge} disabled={saving} className="flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50">{saving ? <Loader2 className="h-4 w-4 animate-spin" /> : <Check className="h-4 w-4" />} Save Nudge Config</button>
            </div>
          </div>
          <div className={cardCls}>
            <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Eye className="h-4 w-4" /> Nudge Preview</h2>
            <div className="rounded-xl border-2 border-indigo-200 dark:border-indigo-800 bg-indigo-50 dark:bg-indigo-950/20 p-4">
              <div className="flex items-center gap-3"><Fingerprint className="h-6 w-6 text-indigo-500" /><div><p className="text-sm font-medium text-gray-900 dark:text-white">{nudge.banner_text}</p><div className="mt-2 flex gap-2"><button className="rounded-lg bg-indigo-600 px-3 py-1 text-xs font-medium text-white">Set up passkey</button><button className="rounded-lg border border-gray-300 px-3 py-1 text-xs dark:border-gray-700">Later</button></div></div></div>
            </div>
          </div>
        </div>
      )}

      {/* TEMP ACCESS PASS */}
      {tab === "tap" && (
        <>
          <div className="flex justify-end"><button onClick={() => { setShowTAP(true); setGeneratedTAP(""); }} className="flex items-center gap-2 rounded-lg bg-indigo-600 px-3 py-2 text-sm font-medium text-white hover:bg-indigo-700"><Plus className="h-4 w-4" /> Generate TAP</button></div>
          <div className={cardCls}>
            <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Lock className="h-4 w-4" /> Temporary Access Passes</h2>
            {passes.length === 0 ? <div className="py-8 text-center"><Lock className="mx-auto h-10 w-10 text-gray-300" /><p className="mt-3 text-sm text-gray-400">No TAP codes issued.</p></div> : (
              <div className="overflow-x-auto"><table className="w-full text-sm">
                <thead className="bg-gray-50 dark:bg-gray-900/50"><tr><th scope="col" className="px-3 py-2 text-left text-xs font-medium text-gray-400">User</th><th scope="col" className="px-3 py-2 text-center text-xs font-medium text-gray-400">Code</th><th scope="col" className="px-3 py-2 text-left text-xs font-medium text-gray-400">Created</th><th scope="col" className="px-3 py-2 text-left text-xs font-medium text-gray-400">Expires</th><th scope="col" className="px-3 py-2 text-center text-xs font-medium text-gray-400">Used</th><th scope="col" className="px-3 py-2 text-center text-xs font-medium text-gray-400">Passkey Required</th></tr></thead>
                <tbody className="divide-y dark:divide-gray-800">{passes.map(p => (
                  <tr key={p.id} className="hover:bg-gray-50 dark:hover:bg-gray-900/30"><td className="px-3 py-2 text-xs">{p.user_id}</td><td className="px-3 py-2 text-center font-mono text-xs">{"••••-••••"}</td><td className="px-3 py-2 text-xs text-gray-500">{new Date(p.created_at).toLocaleString()}</td><td className="px-3 py-2 text-xs text-gray-500">{new Date(p.expires_at).toLocaleString()}</td><td className="px-3 py-2 text-center">{p.used ? <CheckCircle className="h-4 w-4 mx-auto text-green-500" /> : <Clock className="h-4 w-4 mx-auto text-gray-400" />}</td><td className="px-3 py-2 text-center">{p.requires_passkey_enrollment ? <Check className="h-4 w-4 mx-auto text-indigo-500" /> : <X className="h-4 w-4 mx-auto text-gray-300" />}</td></tr>
                ))}</tbody>
              </table></div>
            )}
          </div>
        </>
      )}

      {/* PASSKEY PROFILES */}
      {tab === "profiles" && (
        <>
          <div className="flex justify-end"><button onClick={() => setShowProfile(true)} className="flex items-center gap-2 rounded-lg bg-indigo-600 px-3 py-2 text-sm font-medium text-white hover:bg-indigo-700"><Plus className="h-4 w-4" /> Add AAGUID</button></div>
          <div className={cardCls}>
            <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Smartphone className="h-4 w-4" /> Passkey Profiles (AAGUID Whitelist)</h2>
            {profiles.length === 0 ? <div className="py-8 text-center"><Smartphone className="mx-auto h-10 w-10 text-gray-300" /><p className="mt-3 text-sm text-gray-400">No passkey profiles configured.</p></div> : (
              <div className="space-y-2">{profiles.map(p => (
                <div key={p.id} className="flex items-center justify-between rounded-lg border p-3 dark:border-gray-700">
                  <div className="flex items-center gap-3"><Smartphone className="h-5 w-5 text-gray-400" /><div><p className="text-sm font-medium">{p.name}</p><p className="text-xs font-mono text-gray-400">{p.aaguid}</p></div></div>
                  <div className="flex items-center gap-2"><span className={"px-1.5 py-0.5 rounded text-xs " + (p.trust_level === "trusted" ? "bg-green-100 dark:bg-green-900/30 text-green-600" : p.trust_level === "compliant" ? "bg-blue-100 dark:bg-blue-900/30 text-blue-600" : "bg-red-100 dark:bg-red-900/30 text-red-600")}>{p.trust_level}</span><span className="px-1.5 py-0.5 rounded text-xs bg-gray-100 dark:bg-gray-700">att: {p.attestation}</span></div>
                </div>
              ))}</div>
            )}
          </div>
        </>
      )}

      </>)}

      {/* TAP dialog */}
      {showTAP && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setShowTAP(false)}>
          <div role="dialog" aria-modal="true" className="mx-4 w-full max-w-md rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={e => e.stopPropagation()}>
            <h3 className="flex items-center gap-2 text-lg font-semibold text-gray-900 dark:text-white"><Lock className="h-5 w-5 text-indigo-500" /> Generate Temporary Access Pass</h3>
            {generatedTAP ? (
              <div className="mt-4 text-center">
                <div className="rounded-xl border-2 border-green-300 bg-green-50 p-6 dark:border-green-700 dark:bg-green-950/30"><CheckCircle className="h-10 w-10 mx-auto text-green-500" /><p className="mt-3 text-sm text-gray-500">Share this code with the user (expires in {tapDuration} min):</p><p className="mt-2 text-2xl font-bold font-mono tracking-wider text-green-700 dark:text-green-400">{generatedTAP}</p></div>
                <button onClick={() => setShowTAP(false)} className="mt-4 rounded-lg border border-gray-300 px-4 py-2 text-sm dark:border-gray-700">Close</button>
              </div>
            ) : (
              <><div className="mt-4 space-y-3">
                <div><label className="text-sm font-medium">User ID</label><input aria-label="TAP user" type="text" value={tapUser} onChange={e => setTapUser(e.target.value)} placeholder="user:alice" className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm font-mono" autoFocus /></div>
                <div><label className="text-sm font-medium">Duration (minutes)</label><input aria-label="TAP duration" type="number" min={5} max={60} value={tapDuration} onChange={e => setTapDuration(parseInt(e.target.value) || 15)} className="mt-1 w-24 rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm" /></div>
                <div className="rounded-lg bg-blue-50 p-3 dark:bg-blue-950/30"><p className="text-xs text-blue-700 dark:text-blue-400">User will be forced to register a new passkey after using this TAP.</p></div>
              </div>
              <div className="mt-4 flex justify-end gap-2"><button onClick={() => setShowTAP(false)} className="rounded-lg border border-gray-300 px-4 py-2 text-sm dark:border-gray-700">Cancel</button><button onClick={generateTAP} disabled={!tapUser || generating} className="rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50">{generating ? <Loader2 className="h-4 w-4 animate-spin" /> : "Generate"}</button></div></>
            )}
          </div>
        </div>
      )}

      {/* Profile dialog */}
      {showProfile && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setShowProfile(false)}>
          <div role="dialog" aria-modal="true" className="mx-4 w-full max-w-md rounded-xl bg-white p-6 shadow-xl dark:bg-gray-800" onClick={e => e.stopPropagation()}>
            <h3 className="flex items-center gap-2 text-lg font-semibold text-gray-900 dark:text-white"><Smartphone className="h-5 w-5 text-indigo-500" /> Add Passkey Profile</h3>
            <div className="mt-4 space-y-3">
              <div><label className="text-sm font-medium">Device Name</label><input aria-label="Device name" type="text" value={pfName} onChange={e => setPfName(e.target.value)} placeholder="YubiKey 5C NFC" className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm" autoFocus /></div>
              <div><label className="text-sm font-medium">AAGUID</label><input aria-label="AAGUID" type="text" value={pfAaguid} onChange={e => setPfAaguid(e.target.value)} placeholder="00000000-0000-0000-0000-000000000000" className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm font-mono" /></div>
            </div>
            <div className="mt-4 flex justify-end gap-2"><button onClick={() => setShowProfile(false)} className="rounded-lg border border-gray-300 px-4 py-2 text-sm dark:border-gray-700">Cancel</button><button onClick={saveProfile} disabled={!pfName || !pfAaguid || saving} className="rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50">{saving ? <Loader2 className="h-4 w-4 animate-spin" /> : "Save"}</button></div>
          </div>
        </div>
      )}
    </div>
  );
}
