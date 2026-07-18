"use client";

import { useState, useCallback, useEffect } from "react";
import {
  Shield, Loader2, AlertCircle, X, RefreshCw, Plus, Check, CheckCircle,
  XCircle, Settings, Zap, Activity, Eye, TestTube, Sliders,
  ChevronRight, ArrowRight, AlertTriangle, TrendingUp, Lock, Globe,
  Smartphone, Clock, KeyRound,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";
import { authHeader } from "@/lib/auth-helpers";

const TENANT_ID = "00000000-0000-0000-0000-000000000001";

interface RiskSignal {
  id: string; name: string; icon: typeof Globe; weight: number;
  enabled: boolean; sensitivity: number; cold_start_minutes: number;
  description: string;
}

interface AALThreshold {
  range: string; min: number; max: number; aal: "AAL1" | "AAL2" | "AAL3"; action: string; color: string;
}

interface StepUpRule {
  id: string; condition: string; action: "require_mfa" | "require_webauthn" | "allow" | "deny" | "block"; order: number;
}

interface SimResult {
  risk_score: number; risk_level: string; aal_required: string;
  actions: string[]; signal_breakdown: { signal: string; contribution: number }[];
}

const DEFAULT_SIGNALS: RiskSignal[] = [
  { id: "geo_velocity", name: "Geo Velocity", icon: Globe, weight: 25, enabled: true, sensitivity: 70, cold_start_minutes: 30, description: "Impossible travel detection based on login distance/time" },
  { id: "device_trust", name: "Device Trust", icon: Smartphone, weight: 20, enabled: true, sensitivity: 60, cold_start_minutes: 0, description: "Known device fingerprint + posture check" },
  { id: "ip_rep", name: "IP Reputation", icon: Shield, weight: 20, enabled: true, sensitivity: 50, cold_start_minutes: 0, description: "External threat intel + ASN filtering" },
  { id: "time_anomaly", name: "Time of Day", icon: Clock, weight: 15, enabled: true, sensitivity: 40, cold_start_minutes: 60, description: "Off-hours login anomaly detection" },
  { id: "failed_attempts", name: "Failed Attempts", icon: KeyRound, weight: 20, enabled: true, sensitivity: 80, cold_start_minutes: 0, description: "Brute force / credential stuffing indicator" },
];

const AAL_THRESHOLDS: AALThreshold[] = [
  { range: "0-30", min: 0, max: 30, aal: "AAL1", action: "Password only — no additional factor", color: "bg-green-500" },
  { range: "31-60", min: 31, max: 60, aal: "AAL2", action: "Require TOTP/SMS MFA factor", color: "bg-yellow-500" },
  { range: "61-100", min: 61, max: 100, aal: "AAL3", action: "Require WebAuthn/hardware key", color: "bg-red-500" },
];

const actionConfig = {
  allow: { color: "text-green-600 bg-green-100 dark:bg-green-900/30", icon: CheckCircle },
  require_mfa: { color: "text-yellow-600 bg-yellow-100 dark:bg-yellow-900/30", icon: Shield },
  require_webauthn: { color: "text-orange-600 bg-orange-100 dark:bg-orange-900/30", icon: Lock },
  deny: { color: "text-red-600 bg-red-100 dark:bg-red-900/30", icon: XCircle },
  block: { color: "text-red-800 bg-red-200 dark:bg-red-900/40", icon: AlertTriangle },
};

type Tab = "matrix" | "thresholds" | "orchestration" | "signals" | "simulator";

export default function AdaptiveAuthPage() {
  const t = useTranslations();
  const [tab, setTab] = useState<Tab>("simulator");
  const [signals, setSignals] = useState<RiskSignal[]>(DEFAULT_SIGNALS);
  const [rules, setRules] = useState<StepUpRule[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  // Simulator
  const [simIp, setSimIp] = useState("192.168.1.100");
  const [simDevice, setSimDevice] = useState("trusted");
  const [simLocation, setSimLocation] = useState("San Francisco");
  const [simPrevLocation, setSimPrevLocation] = useState("San Francisco");
  const [simTime, setSimTime] = useState("14");
  const [simFailed, setSimFailed] = useState(0);
  const [simResult, setSimResult] = useState<SimResult | null>(null);
  const [simulating, setSimulating] = useState(false);

  const loadData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const h = { ...authHeader(), "X-Tenant-ID": TENANT_ID };
      const [sigRes, ruleRes] = await Promise.all([
        fetch("/api/v1/auth/risk-engine/signals", { headers: h }).catch(() => null),
        fetch("/api/v1/auth/risk-engine/rules", { headers: h }).catch(() => null),
      ]);
      if (sigRes?.ok) { const d = await sigRes.json(); if (d.signals?.length) setSignals(d.signals); }
      if (ruleRes?.ok) { const d = await ruleRes.json(); setRules(d.rules || d.items || []); }
    } catch { /* keep defaults */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { loadData(); }, [loadData]);

  const saveSignals = async () => {
    setSaving(true);
    try {
      await fetch("/api/v1/auth/risk-engine/signals", {
        method: "PUT",
        headers: { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": TENANT_ID },
        body: JSON.stringify({ signals }),
      });
    } catch { /* noop */ }
    finally { setSaving(false); }
  };

  const runSimulation = () => {
    setSimulating(true);
    setTimeout(() => {
      let score = 0;
      const breakdown: { signal: string; contribution: number }[] = [];
      // Geo velocity
      if (signals.find(s => s.id === "geo_velocity")?.enabled) {
        let contribution = simLocation !== simPrevLocation ? 30 : 0;
        contribution = Math.round(contribution * (signals.find(s => s.id === "geo_velocity")!.weight / 25));
        score += contribution; breakdown.push({ signal: "Geo Velocity", contribution });
      }
      // Device trust
      if (signals.find(s => s.id === "device_trust")?.enabled) {
        let contribution = simDevice === "unmanaged" ? 25 : simDevice === "new" ? 15 : 0;
        contribution = Math.round(contribution * (signals.find(s => s.id === "device_trust")!.weight / 20));
        score += contribution; breakdown.push({ signal: "Device Trust", contribution });
      }
      // IP rep
      if (signals.find(s => s.id === "ip_rep")?.enabled) {
        let contribution = simIp.startsWith("10.") || simIp.startsWith("192.168.") ? 0 : 15;
        contribution = Math.round(contribution * (signals.find(s => s.id === "ip_rep")!.weight / 20));
        score += contribution; breakdown.push({ signal: "IP Reputation", contribution });
      }
      // Time
      if (signals.find(s => s.id === "time_anomaly")?.enabled) {
        const hour = parseInt(simTime);
        let contribution = (hour < 6 || hour > 22) ? 15 : 0;
        contribution = Math.round(contribution * (signals.find(s => s.id === "time_anomaly")!.weight / 15));
        score += contribution; breakdown.push({ signal: "Time of Day", contribution });
      }
      // Failed attempts
      if (signals.find(s => s.id === "failed_attempts")?.enabled) {
        let contribution = Math.min(simFailed * 5, 30);
        contribution = Math.round(contribution * (signals.find(s => s.id === "failed_attempts")!.weight / 20));
        score += contribution; breakdown.push({ signal: "Failed Attempts", contribution });
      }
      score = Math.min(100, score);
      const aal = score <= 30 ? "AAL1" : score <= 60 ? "AAL2" : "AAL3";
      const actions = score <= 30 ? ["Allow"] : score <= 60 ? ["Require MFA (TOTP)"] : score <= 80 ? ["Require WebAuthn"] : ["Block + Alert SOC"];
      setSimResult({ risk_score: score, risk_level: score <= 30 ? "low" : score <= 60 ? "medium" : score <= 80 ? "high" : "critical", aal_required: aal, actions, signal_breakdown: breakdown });
      setSimulating(false);
    }, 800);
  };

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><Activity className="h-6 w-6 text-indigo-500" /> Adaptive Auth Choreography</h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">Risk-based dynamic authentication — signal weighting, AAL thresholds, step-up orchestration, and simulation.</p>
        </div>
        <button onClick={loadData} disabled={loading} aria-label="Refresh" className="flex items-center gap-2 rounded-lg border border-gray-300 px-3 py-2 text-sm text-gray-600 hover:bg-gray-50 dark:border-gray-700 dark:text-gray-300 dark:hover:bg-gray-800"><RefreshCw className={"h-4 w-4 " + (loading ? "animate-spin" : "")} /> Refresh</button>
      </div>

      {error && <div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400"><AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} aria-label="Dismiss" className="ml-auto"><X className="h-4 w-4" /></button></div>}

      <div className="flex gap-1 border-b border-gray-200 dark:border-gray-700 overflow-x-auto">
        {([
          { id: "simulator" as Tab, label: "Simulator", icon: TestTube },
          { id: "matrix" as Tab, label: "Risk Matrix", icon: Sliders },
          { id: "thresholds" as Tab, label: "AAL Thresholds", icon: TrendingUp },
          { id: "orchestration" as Tab, label: "Step-Up Tree", icon: Zap },
          { id: "signals" as Tab, label: "Signal Sources", icon: Settings },
        ]).map(tb => { const Icon = tb.icon; return (
          <button key={tb.id} onClick={() => setTab(tb.id)} aria-pressed={tab === tb.id} className={"flex items-center gap-1.5 border-b-2 px-4 py-2 text-sm font-medium transition whitespace-nowrap " + (tab === tb.id ? "border-indigo-600 text-indigo-600 dark:text-indigo-400" : "border-transparent text-gray-500 hover:text-gray-700 dark:hover:text-gray-300")}><Icon className="h-4 w-4" /> {tb.label}</button>
        ); })}
      </div>

      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-indigo-500" /></div> : (<>

      {/* SIMULATOR */}
      {tab === "simulator" && (
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
          <div className={cardCls}>
            <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><TestTube className="h-4 w-4" /> Context Input</h2>
            <div className="space-y-3">
              <div><label className="text-sm font-medium">IP Address</label><input aria-label="IP" type="text" value={simIp} onChange={e => setSimIp(e.target.value)} className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm font-mono" /></div>
              <div><label className="text-sm font-medium">Device Trust</label><select aria-label="Device" value={simDevice} onChange={e => setSimDevice(e.target.value)} className="mt-1 block w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm"><option value="trusted">Trusted (known)</option><option value="new">New</option><option value="unmanaged">Unmanaged</option></select></div>
              <div className="grid grid-cols-2 gap-3">
                <div><label className="text-sm font-medium">Current Location</label><input aria-label="Location" type="text" value={simLocation} onChange={e => setSimLocation(e.target.value)} className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm" /></div>
                <div><label className="text-sm font-medium">Previous Location</label><input aria-label="Prev location" type="text" value={simPrevLocation} onChange={e => setSimPrevLocation(e.target.value)} className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm" /></div>
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div><label className="text-sm font-medium">Hour (0-23)</label><input aria-label="Hour" type="number" min={0} max={23} value={simTime} onChange={e => setSimTime(e.target.value)} className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm" /></div>
                <div><label className="text-sm font-medium">Failed Attempts</label><input aria-label="Failed" type="number" min={0} value={simFailed} onChange={e => setSimFailed(parseInt(e.target.value) || 0)} className="mt-1 w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm" /></div>
              </div>
              <button onClick={runSimulation} disabled={simulating} className="flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50">{simulating ? <Loader2 className="h-4 w-4 animate-spin" /> : <TestTube className="h-4 w-4" />} Evaluate Risk</button>
            </div>
          </div>
          <div className={cardCls}>
            <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Eye className="h-4 w-4" /> Risk Assessment</h2>
            {simResult ? (
              <div className="space-y-4">
                {/* Score gauge */}
                <div className="text-center">
                  <div className="mx-auto w-32 h-32 relative"><svg width="128" height="128" viewBox="0 0 128 128" className="-rotate-90"><circle cx="64" cy="64" r="52" fill="none" stroke="#e5e7eb" strokeWidth="8" className="dark:stroke-gray-700" /><circle cx="64" cy="64" r="52" fill="none" stroke={simResult.risk_score <= 30 ? "#16a34a" : simResult.risk_score <= 60 ? "#eab308" : simResult.risk_score <= 80 ? "#f97316" : "#dc2626"} strokeWidth="8" strokeLinecap="round" strokeDasharray={`${(simResult.risk_score / 100) * 327} 327`} /></svg><div className="absolute inset-0 flex flex-col items-center justify-center"><span className={"text-3xl font-bold " + (simResult.risk_score <= 30 ? "text-green-600" : simResult.risk_score <= 60 ? "text-yellow-600" : simResult.risk_score <= 80 ? "text-orange-600" : "text-red-600")}>{simResult.risk_score}</span><span className="text-xs text-gray-400">risk</span></div></div>
                </div>
                <div className="flex items-center justify-center gap-2"><span className="text-sm font-medium capitalize">{simResult.risk_level}</span><ArrowRight className="h-4 w-4 text-gray-400" /><span className={"px-3 py-1 rounded-lg font-bold text-sm " + (simResult.aal_required === "AAL1" ? "bg-green-100 text-green-700 dark:bg-green-900/30" : simResult.aal_required === "AAL2" ? "bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30" : "bg-red-100 text-red-700 dark:bg-red-900/30")}>{simResult.aal_required}</span></div>
                {/* Signal breakdown */}
                <div className="space-y-1">{simResult.signal_breakdown.map((s: any, i: number) => (
                  <div key={i} className="flex items-center gap-2"><span className="text-xs text-gray-500 flex-1">{s.signal}</span><div className="w-24 h-2 overflow-hidden rounded-full bg-gray-200 dark:bg-gray-700"><div className="h-full rounded-full bg-indigo-500" style={{ width: `${s.contribution * 3}%` }} /></div><span className="text-xs font-mono w-8 text-right">{s.contribution}</span></div>
                ))}</div>
                {/* Actions */}
                <div className="rounded-lg border p-3 dark:border-gray-700"><p className="text-xs font-semibold uppercase text-gray-400 mb-1">Actions Executed</p>{simResult.actions.map((a: any, i: number) => <div key={i} className="flex items-center gap-2 text-sm"><ChevronRight className="h-3 w-3 text-indigo-400" /><span>{a}</span></div>)}</div>
              </div>
            ) : <div className="py-8 text-center"><TestTube className="mx-auto h-10 w-10 text-gray-300" /><p className="mt-3 text-sm text-gray-400">Configure context and evaluate risk.</p></div>}
          </div>
        </div>
      )}

      {/* RISK MATRIX */}
      {tab === "matrix" && (
        <div className={cardCls}>
          <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Sliders className="h-4 w-4" /> Risk Signal Weight Matrix</h2>
          <div className="space-y-3">{signals.map(sig => { const SIcon = sig.icon; return (
            <div key={sig.id} className="flex items-center gap-4 rounded-lg border p-3 dark:border-gray-700">
              <SIcon className={"h-5 w-5 shrink-0 " + (sig.enabled ? "text-indigo-500" : "text-gray-300")} />
              <div className="flex-1"><div className="flex items-center justify-between"><span className="text-sm font-medium">{sig.name}</span><span className="text-xs text-gray-400">Weight: {sig.weight}%</span></div><p className="text-xs text-gray-400">{sig.description}</p></div>
              <div className="w-32"><input aria-label={`${sig.name} weight`} type="range" min={0} max={50} value={sig.weight} onChange={e => setSignals(prev => prev.map(s => s.id === sig.id ? { ...s, weight: parseInt(e.target.value) } : s))} className="w-full accent-indigo-600" /></div>
              <span className="text-xs font-bold w-8 text-right">{sig.weight}</span>
              <button onClick={() => setSignals(prev => prev.map(s => s.id === sig.id ? { ...s, enabled: !s.enabled } : s))} aria-pressed={sig.enabled} className={"rounded-lg px-2 py-1 text-xs font-medium " + (sig.enabled ? "bg-green-50 text-green-700 dark:bg-green-950/20" : "bg-gray-100 dark:bg-gray-800 text-gray-400")}>{sig.enabled ? "On" : "Off"}</button>
            </div>
          ); })}</div>
          <div className="mt-3 flex items-center justify-between"><span className="text-xs text-gray-400">Total weight: {signals.reduce((a, s) => a + (s.enabled ? s.weight : 0), 0)}%</span><button onClick={saveSignals} disabled={saving} className="flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50">{saving ? <Loader2 className="h-4 w-4 animate-spin" /> : <Check className="h-4 w-4" />} Save Weights</button></div>
        </div>
      )}

      {/* AAL THRESHOLDS */}
      {tab === "thresholds" && (
        <div className="space-y-4">
          <div className={cardCls}>
            <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><TrendingUp className="h-4 w-4" /> Risk → AAL Mapping</h2>
            <div className="space-y-2">{AAL_THRESHOLDS.map(t => (
              <div key={t.aal} className="flex items-center gap-3 rounded-lg border p-4 dark:border-gray-700">
                <div className={"h-8 w-8 rounded-lg flex items-center justify-center text-white text-xs font-bold " + t.color}>{t.aal.replace("AAL", "")}</div>
                <div className="flex-1"><div className="flex items-center gap-2"><span className="font-semibold text-sm">{t.aal}</span><span className="text-xs text-gray-400">Risk: {t.range}</span></div><p className="text-xs text-gray-500">{t.action}</p></div>
                <div className="w-32 h-3 overflow-hidden rounded-full bg-gray-200 dark:bg-gray-700"><div className={"h-full " + t.color} style={{ width: `${t.max}%` }} /></div>
              </div>
            ))}</div>
          </div>
          <div className={cardCls}><h3 className="mb-3 text-sm font-semibold uppercase text-gray-400">Per-Application Overrides</h3><p className="text-sm text-gray-400">Override default AAL thresholds for specific applications (e.g., financial app requires minimum AAL2).</p></div>
        </div>
      )}

      {/* STEP-UP ORCHESTRATION */}
      {tab === "orchestration" && (
        <div className={cardCls}>
          <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Zap className="h-4 w-4" /> Step-Up/Step-Down Decision Tree</h2>
          {rules.length === 0 ? (
            <div className="space-y-3">
              <div className="flex items-center gap-3 rounded-lg border p-3 dark:border-gray-700"><span className="rounded-lg bg-green-100 dark:bg-green-900/30 px-2 py-1 text-xs font-bold text-green-700">Risk ≤ 30</span><ArrowRight className="h-4 w-4 text-gray-400" /><span className="text-sm">Allow login (AAL1)</span></div>
              <div className="flex items-center gap-3 rounded-lg border p-3 dark:border-gray-700"><span className="rounded-lg bg-yellow-100 dark:bg-yellow-900/30 px-2 py-1 text-xs font-bold text-yellow-700">31-60</span><ArrowRight className="h-4 w-4 text-gray-400" /><Shield className="h-4 w-4 text-yellow-500" /><span className="text-sm">Require MFA (TOTP/SMS)</span></div>
              <div className="flex items-center gap-3 rounded-lg border p-3 dark:border-gray-700"><span className="rounded-lg bg-orange-100 dark:bg-orange-900/30 px-2 py-1 text-xs font-bold text-orange-700">61-80</span><ArrowRight className="h-4 w-4 text-gray-400" /><Lock className="h-4 w-4 text-orange-500" /><span className="text-sm">Require WebAuthn (hardware key)</span></div>
              <div className="flex items-center gap-3 rounded-lg border border-red-300 dark:border-red-700 p-3"><span className="rounded-lg bg-red-100 dark:bg-red-900/30 px-2 py-1 text-xs font-bold text-red-700">81-100</span><ArrowRight className="h-4 w-4 text-gray-400" /><AlertTriangle className="h-4 w-4 text-red-500" /><span className="text-sm">Block + Alert SOC</span></div>
            </div>
          ) : (
            <div className="space-y-2">{rules.map(r => { const cfg = actionConfig[r.action]; const RIcon = cfg.icon; return (
              <div key={r.id} className="flex items-center gap-3 rounded-lg border p-3 dark:border-gray-700"><span className="font-mono text-xs text-gray-500 flex-1">{r.condition}</span><ArrowRight className="h-4 w-4 text-gray-400" /><span className={"flex items-center gap-1 px-2 py-1 rounded text-xs font-medium " + cfg.color}><RIcon className="h-3 w-3" /> {r.action}</span></div>
            ); })}</div>
          )}
        </div>
      )}

      {/* SIGNAL SOURCES */}
      {tab === "signals" && (
        <div className={cardCls}>
          <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Settings className="h-4 w-4" /> Signal Source Configuration</h2>
          <div className="space-y-3">{signals.map(sig => { const SIcon = sig.icon; return (
            <div key={sig.id} className="rounded-lg border p-3 dark:border-gray-700">
              <div className="flex items-center justify-between"><div className="flex items-center gap-2"><SIcon className={"h-5 w-5 " + (sig.enabled ? "text-indigo-500" : "text-gray-300")} /><span className="font-medium text-sm">{sig.name}</span></div><button onClick={() => setSignals(prev => prev.map(s => s.id === sig.id ? { ...s, enabled: !s.enabled } : s))} aria-pressed={sig.enabled} className={"rounded-lg px-2 py-1 text-xs font-medium " + (sig.enabled ? "bg-green-50 text-green-700 dark:bg-green-950/20" : "bg-gray-100 dark:bg-gray-800 text-gray-400")}>{sig.enabled ? "Enabled" : "Disabled"}</button></div>
              <div className="mt-2 grid grid-cols-2 gap-4"><div><label className="text-xs font-medium text-gray-400">Sensitivity: {sig.sensitivity}%</label><input aria-label={`${sig.name} sensitivity`} type="range" min={0} max={100} value={sig.sensitivity} onChange={e => setSignals(prev => prev.map(s => s.id === sig.id ? { ...s, sensitivity: parseInt(e.target.value) } : s))} className="w-full accent-indigo-600" /></div><div><label className="text-xs font-medium text-gray-400">Cold Start: {sig.cold_start_minutes}min</label><input aria-label={`${sig.name} cold start`} type="range" min={0} max={120} value={sig.cold_start_minutes} onChange={e => setSignals(prev => prev.map(s => s.id === sig.id ? { ...s, cold_start_minutes: parseInt(e.target.value) } : s))} className="w-full accent-indigo-600" /></div></div>
            </div>
          ); })}</div>
          <button onClick={saveSignals} disabled={saving} className="mt-4 flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50">{saving ? <Loader2 className="h-4 w-4 animate-spin" /> : <Check className="h-4 w-4" />} Save Configuration</button>
        </div>
      )}

      </>)}
    </div>
  );
}
