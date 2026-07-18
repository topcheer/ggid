"use client";
import { useState, useCallback, useEffect } from "react";
import {
  Sliders, Loader2, AlertCircle, X, RefreshCw, Save, RotateCcw,
  Shield, Gauge, ChevronRight, CheckCircle2, Zap, Building2,
  Smartphone, MapPin, Wifi, Activity, Clock,
} from "lucide-react";
import { authHeader } from "@/lib/auth-helpers";
import { useTranslations } from "@/lib/i18n";

const TENANT_ID = "00000000-0000-0000-0000-000000000001";

interface SignalDef { id: string; name: string; category: string; weight: number; default: number; }
interface RiskPolicy { allow_threshold: number; step_up_threshold: number; strong_threshold: number; weights: Record<string, number>; }

type Tab = "weights" | "thresholds" | "tenants" | "catalog";

const CATEGORY_ICONS: Record<string, typeof Smartphone> = {
  device: Smartphone, geo: MapPin, network: Wifi, behavior: Activity, session: Clock,
};
const CATEGORY_COLORS: Record<string, string> = {
  device: "text-blue-500", geo: "text-purple-500", network: "text-orange-500", behavior: "text-green-500", session: "text-cyan-500",
};

// 26 signals from backend registry
const ALL_SIGNALS: SignalDef[] = [
  { id: "device_posture", name: "Device Posture Score", category: "device", weight: 0.15, default: 0 },
  { id: "device_managed", name: "Managed Device", category: "device", weight: 0.10, default: 0 },
  { id: "device_encrypted", name: "Disk Encryption", category: "device", weight: 0.08, default: 0 },
  { id: "device_jailbreak", name: "Jailbreak/Root Detected", category: "device", weight: 0.20, default: 0 },
  { id: "device_compliant_os", name: "OS Compliance", category: "device", weight: 0.07, default: 0 },
  { id: "device_trust_score", name: "Device Trust Score", category: "device", weight: 0.10, default: 0 },
  { id: "geo_impossible_travel", name: "Impossible Travel", category: "geo", weight: 0.25, default: 0 },
  { id: "geo_high_risk_country", name: "High-Risk Country", category: "geo", weight: 0.15, default: 0 },
  { id: "geo_new_location", name: "New Login Location", category: "geo", weight: 0.08, default: 0 },
  { id: "geo_vpn_proxy", name: "VPN/Proxy Detected", category: "geo", weight: 0.12, default: 0 },
  { id: "geo_velocity", name: "Geo Velocity", category: "geo", weight: 0.10, default: 0 },
  { id: "network_asn_trust", name: "ASN Trust Score", category: "network", weight: 0.08, default: 0 },
  { id: "network_ip_reputation", name: "IP Reputation", category: "network", weight: 0.20, default: 0 },
  { id: "network_tor_exit", name: "Tor Exit Node", category: "network", weight: 0.25, default: 0 },
  { id: "network_datacenter", name: "Datacenter IP", category: "network", weight: 0.15, default: 0 },
  { id: "network_port_scan", name: "Port Scan Activity", category: "network", weight: 0.12, default: 0 },
  { id: "behavior_login_velocity", name: "Login Velocity", category: "behavior", weight: 0.15, default: 0 },
  { id: "behavior_failed_attempts", name: "Failed Attempts", category: "behavior", weight: 0.20, default: 0 },
  { id: "behavior_off_hours", name: "Off-Hours Access", category: "behavior", weight: 0.10, default: 0 },
  { id: "behavior_new_user_agent", name: "New User Agent", category: "behavior", weight: 0.05, default: 0 },
  { id: "behavior_api_anomaly", name: "API Usage Anomaly", category: "behavior", weight: 0.12, default: 0 },
  { id: "behavior_resource_spike", name: "Resource Access Spike", category: "behavior", weight: 0.08, default: 0 },
  { id: "session_concurrent", name: "Concurrent Sessions", category: "session", weight: 0.10, default: 0 },
  { id: "session_duration", name: "Session Duration Anomaly", category: "session", weight: 0.08, default: 0 },
  { id: "session_token_age", name: "Token Age", category: "session", weight: 0.05, default: 0 },
  { id: "session_privilege_use", name: "Privileged Action", category: "session", weight: 0.15, default: 0 },
];

function DecisionGauge({ allow, stepUp, strong }: { allow: number; stepUp: number; strong: number }) {
  const zones = [
    { label: "Allow", from: 0, to: allow, color: "#22c55e" },
    { label: "Step-up", from: allow, to: stepUp, color: "#eab308" },
    { label: "Strong Auth", from: stepUp, to: strong, color: "#f97316" },
    { label: "Block", from: strong, to: 100, color: "#ef4444" },
  ];
  return (
    <div className="flex h-6 w-full overflow-hidden rounded-full">
      {zones.map(z => <div key={z.label} className="flex items-center justify-center text-xs font-bold text-white" style={{ width: `${z.to - z.from}%`, backgroundColor: z.color }}>{z.to - z.from > 10 && z.label}</div>)}
    </div>
  );
}

export default function RiskPolicyPage() {
  const t = useTranslations();
  const [tab, setTab] = useState<Tab>("weights");
  const [signals, setSignals] = useState<SignalDef[]>(ALL_SIGNALS);
  const [policy, setPolicy] = useState<RiskPolicy>({ allow_threshold: 30, step_up_threshold: 60, strong_threshold: 85, weights: {} });
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const [selectedTenant, setSelectedTenant] = useState(TENANT_ID);

  const h = { ...authHeader(), "X-Tenant-ID": TENANT_ID };
  const H = { ...authHeader(), "Content-Type": "application/json", "X-Tenant-ID": TENANT_ID };
  const card = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  const loadData = useCallback(async () => {
    setLoading(true);
    try {
      const [sRes, pRes] = await Promise.all([
        fetch("/api/v1/risk/signals", { headers: h }).catch(() => null),
        fetch(`/api/v1/risk/policies/${TENANT_ID}`, { headers: h }).catch(() => null),
      ]);
      if (sRes?.ok) { const d = await sRes.json(); if (d.signals?.length) setSignals(d.signals); }
      if (pRes?.ok) { const d = await pRes.json(); setPolicy(d); }
    } catch { setError(t("riskPolicy.loadError")); }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { loadData(); }, [loadData]);

  const updateWeight = (id: string, weight: number) => {
    setSignals(prev => prev.map(s => s.id === id ? { ...s, weight } : s));
  };

  const updateThreshold = (key: keyof RiskPolicy, val: number) => {
    setPolicy(prev => ({ ...prev, [key]: val }));
  };

  const save = async () => {
    setSaving(true);
    try {
      const weights: Record<string, number> = {};
      signals.forEach(s => weights[s.id] = s.weight);
      await fetch(`/api/v1/risk/policies/${selectedTenant}`, { method: "PUT", headers: H, body: JSON.stringify({ ...policy, weights }) });
    } catch { setError(t("riskPolicy.saveError")); }
    finally { setSaving(false); }
  };

  const reset = () => {
    setSignals(ALL_SIGNALS);
    setPolicy({ allow_threshold: 30, step_up_threshold: 60, strong_threshold: 85, weights: {} });
  };

  const categories = [...new Set(signals.map(s => s.category))];
  const totalWeight = signals.reduce((a: any, s: any) => a + s.weight, 0);

  // Sample score preview
  const sampleScore = Math.round(signals.filter(s => ["geo_impossible_travel", "network_ip_reputation", "behavior_failed_attempts", "device_jailbreak"].includes(s.id)).reduce((a: any, s: any) => a + s.weight * 100, 0));

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white"><Sliders className="h-6 w-6 text-orange-500" /> {t("riskPolicy.title")}</h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{t("riskPolicy.subtitle")}</p>
        </div>
        <div className="flex items-center gap-2">
          <button onClick={reset} className="flex items-center gap-1 rounded-lg border border-gray-300 px-3 py-1.5 text-xs dark:border-gray-700"><RotateCcw className="h-3 w-3" /> {t("riskPolicy.reset")}</button>
          <button onClick={save} disabled={saving} className="flex items-center gap-1 rounded-lg bg-orange-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-orange-700 disabled:opacity-50">{saving ? <Loader2 className="h-3 w-3 animate-spin" /> : <Save className="h-3 w-3" />} {t("riskPolicy.save")}</button>
        </div>
      </div>

      {error && (
        <div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400">
          <AlertCircle className="h-4 w-4 shrink-0" />{error}<button onClick={() => setError(null)} aria-label="Dismiss" className="ml-auto"><X className="h-4 w-4" /></button>
        </div>
      )}

      <div className="flex gap-1 border-b border-gray-200 dark:border-gray-700 overflow-x-auto">
        {([
          { id: "weights" as Tab, label: t("riskPolicy.weights"), icon: Sliders },
          { id: "thresholds" as Tab, label: t("riskPolicy.thresholds"), icon: Gauge },
          { id: "tenants" as Tab, label: t("riskPolicy.tenantOverride"), icon: Building2 },
          { id: "catalog" as Tab, label: t("riskPolicy.catalog"), icon: Zap },
        ]).map(tb => {
          const Icon = tb.icon;
          return (
            <button key={tb.id} onClick={() => setTab(tb.id)} aria-pressed={tab === tb.id}
              className={`flex items-center gap-1.5 border-b-2 px-4 py-2 text-sm font-medium transition whitespace-nowrap ${tab === tb.id ? "border-orange-600 text-orange-600 dark:text-orange-400" : "border-transparent text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"}`}>
              <Icon className="h-4 w-4" /> {tb.label}
            </button>
          );
        })}
      </div>

      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-orange-500" /></div> : (<>

      {/* ════ WEIGHTS ════ */}
      {tab === "weights" && (
        <div className="space-y-6">
          {/* Sample preview */}
          <div className={card}>
            <div className="flex items-center justify-between">
              <div><p className="text-xs text-gray-400">{t("riskPolicy.sampleScore")}</p><p className={`text-3xl font-bold ${sampleScore >= 85 ? "text-red-600" : sampleScore >= 60 ? "text-orange-600" : sampleScore >= 30 ? "text-yellow-600" : "text-green-600"}`}>{sampleScore}/100</p></div>
              <div className="text-right"><p className="text-xs text-gray-400">{t("riskPolicy.totalWeight")}</p><p className="text-lg font-bold">{totalWeight.toFixed(2)}</p></div>
            </div>
          </div>

          {categories.map(cat => {
            const catSignals = signals.filter(s => s.category === cat);
            const CatIcon = CATEGORY_ICONS[cat] || Shield;
            return (
              <div key={cat} className={card}>
                <h3 className="mb-3 flex items-center gap-2 text-sm font-semibold"><CatIcon className={`h-4 w-4 ${CATEGORY_COLORS[cat] || "text-gray-400"}`} /> <span className="capitalize">{cat}</span> <span className="text-xs text-gray-400">({catSignals.length})</span></h3>
                <div className="space-y-3">
                  {catSignals.map(s => (
                    <div key={s.id}>
                      <div className="flex items-center justify-between mb-1">
                        <span className="text-xs">{s.name}</span>
                        <span className="text-xs font-mono font-bold">{s.weight.toFixed(2)}</span>
                      </div>
                      <input type="range" min={0} max={1} step={0.01} value={s.weight} onChange={e => updateWeight(s.id, parseFloat(e.target.value))} className="w-full accent-orange-500" aria-label={s.name} />
                    </div>
                  ))}
                </div>
              </div>
            );
          })}
        </div>
      )}

      {/* ════ THRESHOLDS ════ */}
      {tab === "thresholds" && (
        <div className="space-y-6">
          <div className={card}>
            <h3 className="mb-4 text-sm font-semibold uppercase text-gray-400">{t("riskPolicy.decisionZones")}</h3>
            <DecisionGauge allow={policy.allow_threshold} stepUp={policy.step_up_threshold} strong={policy.strong_threshold} />
            <div className="mt-2 flex justify-between text-xs text-gray-400">
              <span>0</span><span>{policy.allow_threshold}</span><span>{policy.step_up_threshold}</span><span>{policy.strong_threshold}</span><span>100</span>
            </div>
          </div>
          <div className={card}>
            <h3 className="mb-4 text-sm font-semibold uppercase text-gray-400">{t("riskPolicy.thresholdConfig")}</h3>
            <div className="space-y-4">
              {([
                { key: "allow_threshold" as keyof RiskPolicy, label: t("riskPolicy.allowBoundary"), desc: "Below: allow access", color: "accent-green-500" },
                { key: "step_up_threshold" as keyof RiskPolicy, label: t("riskPolicy.stepUpBoundary"), desc: "Below: require MFA", color: "accent-yellow-500" },
                { key: "strong_threshold" as keyof RiskPolicy, label: t("riskPolicy.strongBoundary"), desc: "Below: strong auth, Above: block", color: "accent-red-500" },
              ]).map(item => (
                <div key={item.key}>
                  <div className="flex items-center justify-between mb-1"><span className="text-sm font-medium">{item.label}</span><span className="text-sm font-mono font-bold">{policy[item.key] as number}</span></div>
                  <input type="range" min={0} max={100} value={policy[item.key] as number} onChange={e => updateThreshold(item.key, parseInt(e.target.value))} className={`w-full ${item.color}`} aria-label={item.label} />
                  <p className="text-xs text-gray-400">{item.desc}</p>
                </div>
              ))}
            </div>
          </div>
        </div>
      )}

      {/* ════ TENANT ════ */}
      {tab === "tenants" && (
        <div className={card}>
          <h3 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Building2 className="h-4 w-4" /> {t("riskPolicy.tenantOverride")}</h3>
          <div className="space-y-3">
            <div><label className="text-sm font-medium">{t("riskPolicy.selectTenant")}</label>
              <select value={selectedTenant} onChange={e => setSelectedTenant(e.target.value)} className="mt-1 block w-full rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-3 py-2 text-sm">
                <option value={TENANT_ID}>Default Tenant</option>
                <option value="00000000-0000-0000-0000-000000000002">Enterprise Tenant</option>
                <option value="00000000-0000-0000-0000-000000000003">Trial Tenant</option>
              </select>
            </div>
            <div className="rounded-lg border p-3 dark:border-gray-700">
              <div className="grid grid-cols-3 gap-3 text-center">
                <div><p className="text-xs text-gray-400">{t("riskPolicy.allowBoundary")}</p><p className="text-lg font-bold text-green-600">{policy.allow_threshold}</p></div>
                <div><p className="text-xs text-gray-400">{t("riskPolicy.stepUpBoundary")}</p><p className="text-lg font-bold text-yellow-600">{policy.step_up_threshold}</p></div>
                <div><p className="text-xs text-gray-400">{t("riskPolicy.strongBoundary")}</p><p className="text-lg font-bold text-red-600">{policy.strong_threshold}</p></div>
              </div>
            </div>
            <button onClick={save} disabled={saving} className="flex items-center gap-2 rounded-lg bg-orange-600 px-4 py-2 text-sm font-medium text-white hover:bg-orange-700 disabled:opacity-50">{saving ? <Loader2 className="h-4 w-4 animate-spin" /> : <Save className="h-4 w-4" />} {t("riskPolicy.saveTenant")}</button>
          </div>
        </div>
      )}

      {/* ════ CATALOG ════ */}
      {tab === "catalog" && (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {signals.map(s => {
            const CatIcon = CATEGORY_ICONS[s.category] || Shield;
            return (
              <div key={s.id} className={card + " hover:shadow-md transition"}>
                <div className="flex items-center gap-3 mb-2">
                  <div className={`flex h-9 w-9 items-center justify-center rounded-lg bg-gray-100 dark:bg-gray-700`}><CatIcon className={`h-4 w-4 ${CATEGORY_COLORS[s.category] || "text-gray-400"}`} /></div>
                  <div><h3 className="font-semibold text-sm">{s.name}</h3><p className="text-xs text-gray-400 capitalize">{s.category} · <code className="font-mono">{s.id}</code></p></div>
                </div>
                <div className="flex items-center justify-between text-xs"><span className="text-gray-400">{t("riskPolicy.weight")}</span><span className="font-mono font-bold">{s.weight.toFixed(2)}</span></div>
              </div>
            );
          })}
        </div>
      )}

      </>)}
    </div>
  );
}
