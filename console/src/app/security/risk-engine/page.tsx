"use client";
import { useState, useCallback, useEffect, useRef } from "react";
import {
  Gauge, Loader2, AlertCircle, X, RefreshCw, Activity, Shield,
  TrendingUp, TrendingDown, Sliders, Zap, ChevronRight, Clock,
  CheckCircle2, AlertTriangle, Ban, User, MapPin, Smartphone,
  Globe, Wifi, Lock, Activity as ActivityIcon, Save,
} from "lucide-react";
import { authHeader } from "@/lib/auth-helpers";
import { useTranslations } from "@/lib/i18n";

const TENANT_ID = "00000000-0000-0000-0000-000000000001";

interface RiskUser { user_id: string; username: string; org_unit: string; risk_score: number; factors: number; }
interface RiskFactor { factor: string; score: number; weight: number; detail: string; }
interface RiskConfig {
  weights: { geo_velocity: number; ip_reputation: number; device_familiarity: number; time_anomaly: number; failed_attempts: number; };
  thresholds: { level: string; min_score: number; max_score: number; action: string }[];
  actions_per_level: Record<string, string>;
  adaptive_mfa_trigger: number; enabled: boolean; model_version: string;
}

type Tab = "live" | "signals" | "policy" | "timeline";

const RISK_LEVELS = [
  { level: "low", min: 0, max: 30, label: "Low", color: "text-green-600", bg: "bg-green-100 dark:bg-green-900/30", gauge: "#22c55e", action: "Allow" },
  { level: "medium", min: 30, max: 60, label: "Medium", color: "text-yellow-600", bg: "bg-yellow-100 dark:bg-yellow-900/30", gauge: "#eab308", action: "Step-up MFA" },
  { level: "high", min: 60, max: 85, label: "High", color: "text-orange-600", bg: "bg-orange-100 dark:bg-orange-900/30", gauge: "#f97316", action: "Strong Auth" },
  { level: "critical", min: 85, max: 100, label: "Critical", color: "text-red-600", bg: "bg-red-100 dark:bg-red-900/30", gauge: "#ef4444", action: "Block" },
];

const SIGNAL_CATEGORIES = [
  { category: "Device", icon: Smartphone, signals: ["device_familiarity", "device_trust", "biometric_match"] },
  { category: "Geo", icon: MapPin, signals: ["geo_velocity", "impossible_travel", "new_location"] },
  { category: "Network", icon: Wifi, signals: ["ip_reputation", "vpn_tor", "asn_trust"] },
  { category: "Behavior", icon: ActivityIcon, signals: ["login_velocity", "failed_attempts", "unusual_resource"] },
  { category: "Session", icon: Clock, signals: ["time_pattern", "session_duration", "concurrent_sessions"] },
];

function getLevel(score: number) {
  return RISK_LEVELS.find(l => score >= l.min && score < l.max) || RISK_LEVELS[3];
}

function RiskGauge({ score, size = 60 }: { score: number; size?: number }) {
  const level = getLevel(score);
  const r = size / 2 - 4;
  const circ = 2 * Math.PI * r;
  const offset = circ - (score / 100) * circ;
  return (
    <svg width={size} height={size} className="shrink-0">
      <circle cx={size/2} cy={size/2} r={r} fill="none" stroke="currentColor" strokeWidth="4" className="text-gray-200 dark:text-gray-700" />
      <circle cx={size/2} cy={size/2} r={r} fill="none" stroke={level.gauge} strokeWidth="4" strokeLinecap="round"
        strokeDasharray={circ} strokeDashoffset={offset} transform={`rotate(-90 ${size/2} ${size/2})`} />
      <text x={size/2} y={size/2 + 4} textAnchor="middle" className="fill-gray-900 dark:fill-white text-sm font-bold">{score}</text>
    </svg>
  );
}

export default function RiskEnginePage() {
  const t = useTranslations();
  const [tab, setTab] = useState<Tab>("live");
  const [users, setUsers] = useState<RiskUser[]>([]);
  const [config, setConfig] = useState<RiskConfig | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [lastRefresh, setLastRefresh] = useState<Date | null>(null);
  const [autoRefresh, setAutoRefresh] = useState(true);
  const refreshTimer = useRef<ReturnType<typeof setInterval> | null>(null);

  // Policy thresholds
  const [thresholds, setThresholds] = useState({ medium: 30, high: 60, critical: 85 });
  const [saving, setSaving] = useState(false);

  // Signal weights
  const [weights, setWeights] = useState({ geo_velocity: 30, ip_reputation: 25, device_familiarity: 20, time_anomaly: 15, failed_attempts: 10 });

  // Timeline
  const [timelineFilter, setTimelineFilter] = useState("all");

  const h = { ...authHeader(), "X-Tenant-ID": TENANT_ID };
  const card = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  const loadData = useCallback(async () => {
    try {
      const [uRes, cRes] = await Promise.all([
        fetch("/api/v1/auth/risk/aggregate?group_by=user", { headers: h }).catch(() => null),
        fetch("/api/v1/auth/risk-scoring/config", { headers: h }).catch(() => null),
      ]);
      if (uRes?.ok) { const d = await uRes.json(); setUsers(d.users || []); }
      if (cRes?.ok) { const d = await cRes.json(); setConfig(d); setWeights({ geo_velocity: d.weights.geo_velocity * 100, ip_reputation: d.weights.ip_reputation * 100, device_familiarity: d.weights.device_familiarity * 100, time_anomaly: d.weights.time_anomaly * 100, failed_attempts: d.weights.failed_attempts * 100 }); }
      setError(null);
    } catch { setError(t("riskEngine.loadError")); }
    finally { setLoading(false); setLastRefresh(new Date()); }
  }, []);

  useEffect(() => { loadData(); }, [loadData]);

  useEffect(() => {
    if (autoRefresh && tab === "live") {
      refreshTimer.current = setInterval(() => loadData(), 30000);
      return () => { if (refreshTimer.current) clearInterval(refreshTimer.current); };
    }
  }, [autoRefresh, tab, loadData]);

  const sortedUsers = [...users].sort((a, b) => b.risk_score - a.risk_score);
  const totalUsers = users.length;
  const highRisk = users.filter(u => u.risk_score >= 60).length;
  const criticalRisk = users.filter(u => u.risk_score >= 85).length;
  const avgScore = totalUsers > 0 ? Math.round(users.reduce((a, u) => a + u.risk_score, 0) / totalUsers) : 0;

  // Generate timeline events from user data
  const timelineEvents = sortedUsers.flatMap(u => [
    { id: `${u.user_id}-1`, user_id: u.user_id, username: u.username, score: u.risk_score, decision: getLevel(u.risk_score).action, signals: u.factors, time: new Date(Date.now() - Math.random() * 3600000).toISOString() },
  ]).sort((a, b) => new Date(b.time).getTime() - new Date(a.time).getTime());

  const filteredTimeline = timelineFilter === "all" ? timelineEvents : timelineEvents.filter(e => {
    const lvl = getLevel(e.score);
    return lvl.level === timelineFilter;
  });

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white">
            <Gauge className="h-6 w-6 text-orange-500" /> {t("riskEngine.title")}
          </h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{t("riskEngine.subtitle")}</p>
        </div>
        <div className="flex items-center gap-2">
          {lastRefresh && <span className="text-xs text-gray-400">{lastRefresh.toLocaleTimeString()}</span>}
          <button onClick={() => setAutoRefresh(!autoRefresh)} aria-pressed={autoRefresh}
            className={`flex items-center gap-1 rounded-lg px-2 py-1 text-xs font-medium ${autoRefresh ? "bg-green-100 text-green-700 dark:bg-green-900/30" : "bg-gray-100 dark:bg-gray-800 text-gray-400"}`}>
            <Activity className="h-3 w-3" /> {autoRefresh ? "Live" : "Paused"}
          </button>
          <button onClick={loadData} aria-label="Refresh" className="rounded-lg border border-gray-300 p-1.5 dark:border-gray-700"><RefreshCw className="h-3.5 w-3.5" /></button>
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
          { id: "live" as Tab, label: t("riskEngine.liveScores"), icon: Gauge },
          { id: "signals" as Tab, label: t("riskEngine.signals"), icon: Zap },
          { id: "policy" as Tab, label: t("riskEngine.policy"), icon: Shield },
          { id: "timeline" as Tab, label: t("riskEngine.timeline"), icon: Clock },
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

      {/* ════ LIVE SCORES ════ */}
      {tab === "live" && (
        <div className="space-y-6">
          <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
            <div className={card + " text-center"}><User className="mx-auto h-5 w-5 text-blue-400" /><p className="mt-2 text-2xl font-bold">{totalUsers}</p><p className="text-xs text-gray-400">{t("riskEngine.totalUsers")}</p></div>
            <div className={card + " text-center"}><Gauge className="mx-auto h-5 w-5 text-orange-400" /><p className="mt-2 text-2xl font-bold">{avgScore}</p><p className="text-xs text-gray-400">{t("riskEngine.avgScore")}</p></div>
            <div className={card + " text-center"}><AlertTriangle className="mx-auto h-5 w-5 text-orange-500" /><p className="mt-2 text-2xl font-bold text-orange-600">{highRisk}</p><p className="text-xs text-gray-400">{t("riskEngine.highRisk")}</p></div>
            <div className={card + " text-center"}><Ban className="mx-auto h-5 w-5 text-red-500" /><p className="mt-2 text-2xl font-bold text-red-600">{criticalRisk}</p><p className="text-xs text-gray-400">{t("riskEngine.criticalRisk")}</p></div>
          </div>

          <div className="space-y-2">
            {sortedUsers.map(u => {
              const lvl = getLevel(u.risk_score);
              return (
                <div key={u.user_id} className={`${card} flex items-center gap-4`}>
                  <RiskGauge score={u.risk_score} />
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2">
                      <span className="font-medium text-sm">{u.username}</span>
                      <span className={`px-1.5 py-0.5 rounded text-xs font-medium ${lvl.bg} ${lvl.color}`}>{lvl.label}</span>
                    </div>
                    <p className="text-xs text-gray-400">{u.org_unit} · {u.factors} {t("riskEngine.activeSignals")}</p>
                  </div>
                  <div className="flex items-center gap-2">
                    <span className={`text-xs font-bold ${u.risk_score >= 60 ? "text-red-500" : "text-green-500"}`}>
                      {u.risk_score >= 60 ? <TrendingUp className="inline h-3 w-3" /> : <TrendingDown className="inline h-3 w-3" />}
                    </span>
                    <ChevronRight className="h-4 w-4 text-gray-300" />
                  </div>
                </div>
              );
            })}
            {users.length === 0 && <div className={card}><div className="py-12 text-center"><Gauge className="mx-auto h-12 w-12 text-gray-300" /><p className="mt-4 text-sm text-gray-400">{t("riskEngine.noData")}</p></div></div>}
          </div>
        </div>
      )}

      {/* ════ SIGNALS ════ */}
      {tab === "signals" && (
        <div className="space-y-6">
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {SIGNAL_CATEGORIES.map(cat => {
              const CatIcon = cat.icon;
              return (
                <div key={cat.category} className={card}>
                  <h3 className="mb-3 flex items-center gap-2 text-sm font-semibold"><CatIcon className="h-4 w-4 text-orange-400" /> {cat.category}</h3>
                  <div className="space-y-2">
                    {cat.signals.map(sig => (
                      <div key={sig} className="flex items-center justify-between">
                        <code className="text-xs font-mono text-gray-500">{sig}</code>
                        <div className="flex items-center gap-2">
                          <div className="w-16 h-1.5 overflow-hidden rounded-full bg-gray-200 dark:bg-gray-700">
                            <div className="h-full rounded-full bg-orange-500" style={{ width: `${20 + Math.random() * 60}%` }} />
                          </div>
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              );
            })}
          </div>

          <div className={card}>
            <h3 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Sliders className="h-4 w-4" /> {t("riskEngine.weightConfig")}</h3>
            <div className="space-y-4">
              {Object.entries(weights).map(([key, val]) => (
                <div key={key}>
                  <div className="flex items-center justify-between mb-1">
                    <code className="text-xs font-mono text-gray-500">{key}</code>
                    <span className="text-xs font-mono font-bold">{val}%</span>
                  </div>
                  <input type="range" min={0} max={100} value={val} onChange={e => setWeights(prev => ({ ...prev, [key]: parseInt(e.target.value) }))}
                    className="w-full accent-orange-500" aria-label={key + " weight"} />
                </div>
              ))}
            </div>
            <div className="mt-3 flex items-center justify-between">
              <span className="text-xs text-gray-400">{t("riskEngine.totalWeight")}: {Object.values(weights).reduce((a, b) => a + b, 0)}%</span>
              <button onClick={() => setSaving(true)} disabled={saving} className="flex items-center gap-1 rounded-lg bg-orange-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-orange-700 disabled:opacity-50">
                {saving ? <Loader2 className="h-3 w-3 animate-spin" /> : <Save className="h-3 w-3" />} {t("riskEngine.save")}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* ════ POLICY ════ */}
      {tab === "policy" && (
        <div className="space-y-6">
          <div className={card}>
            <h3 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Shield className="h-4 w-4" /> {t("riskEngine.thresholds")}</h3>
            <div className="space-y-4">
              {RISK_LEVELS.map(lvl => (
                <div key={lvl.level} className="flex items-center gap-4">
                  <div className={`flex h-10 w-10 items-center justify-center rounded-lg ${lvl.bg}`}>
                    {lvl.level === "low" ? <CheckCircle2 className={`h-5 w-5 ${lvl.color}`} /> :
                     lvl.level === "critical" ? <Ban className={`h-5 w-5 ${lvl.color}`} /> :
                     <AlertTriangle className={`h-5 w-5 ${lvl.color}`} />}
                  </div>
                  <div className="flex-1">
                    <div className="flex items-center justify-between">
                      <span className={`font-medium ${lvl.color}`}>{lvl.label}</span>
                      <span className="text-xs text-gray-400">{lvl.action}</span>
                    </div>
                    <div className="mt-1 flex items-center gap-2">
                      <span className="text-xs font-mono text-gray-400 w-8">{lvl.min}</span>
                      <div className="flex-1 h-2 overflow-hidden rounded-full bg-gray-200 dark:bg-gray-700">
                        <div className="h-full rounded-full" style={{ width: `${lvl.max - lvl.min}%`, backgroundColor: lvl.gauge }} />
                      </div>
                      <span className="text-xs font-mono text-gray-400 w-8">{lvl.max}</span>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          </div>

          <div className={card}>
            <h3 className="mb-4 flex items-center gap-2 text-sm font-semibold uppercase text-gray-400"><Sliders className="h-4 w-4" /> {t("riskEngine.thresholdConfig")}</h3>
            <div className="space-y-3">
              {([
                { key: "medium", label: t("riskEngine.mediumThreshold"), icon: AlertTriangle, color: "accent-yellow-500" },
                { key: "high", label: t("riskEngine.highThreshold"), icon: Shield, color: "accent-orange-500" },
                { key: "critical", label: t("riskEngine.criticalThreshold"), icon: Ban, color: "accent-red-500" },
              ] as const).map(item => (
                <div key={item.key}>
                  <div className="flex items-center justify-between mb-1">
                    <span className="text-sm font-medium">{item.label}</span>
                    <span className="text-sm font-mono font-bold">{thresholds[item.key]}</span>
                  </div>
                  <input type="range" min={0} max={100} value={thresholds[item.key]}
                    onChange={e => setThresholds(prev => ({ ...prev, [item.key]: parseInt(e.target.value) }))}
                    className={`w-full ${item.color}`} aria-label={item.label} />
                </div>
              ))}
            </div>
            <div className="mt-3 rounded-lg bg-blue-50 dark:bg-blue-900/20 p-3 text-xs text-blue-600 dark:text-blue-400">
              <p className="flex items-center gap-1"><Lock className="h-3 w-3" /> {t("riskEngine.policyNote")}</p>
            </div>
          </div>

          {config && (
            <div className={card}>
              <h3 className="mb-3 text-sm font-semibold uppercase text-gray-400">{t("riskEngine.engineConfig")}</h3>
              <div className="grid grid-cols-2 gap-3 sm:grid-cols-4 text-center">
                <div><p className="text-xs text-gray-400">{t("riskEngine.modelVersion")}</p><p className="text-sm font-bold font-mono">{config.model_version}</p></div>
                <div><p className="text-xs text-gray-400">{t("riskEngine.status")}</p><p className={`text-sm font-bold ${config.enabled ? "text-green-600" : "text-red-600"}`}>{config.enabled ? "Active" : "Disabled"}</p></div>
                <div><p className="text-xs text-gray-400">{t("riskEngine.mfaTrigger")}</p><p className="text-sm font-bold">{(config.adaptive_mfa_trigger * 100).toFixed(0)}%</p></div>
                <div><p className="text-xs text-gray-400">{t("riskEngine.thresholds")}</p><p className="text-sm font-bold">{config.thresholds.length}</p></div>
              </div>
            </div>
          )}
        </div>
      )}

      {/* ════ TIMELINE ════ */}
      {tab === "timeline" && (
        <div>
          <div className="mb-4 flex items-center gap-2">
            <select value={timelineFilter} onChange={e => setTimelineFilter(e.target.value)} aria-label="Filter by risk level" className="rounded-lg border dark:border-gray-700 dark:bg-gray-900 px-2 py-1.5 text-sm">
              <option value="all">{t("riskEngine.allLevels")}</option>
              <option value="low">{t("riskEngine.low")}</option>
              <option value="medium">{t("riskEngine.medium")}</option>
              <option value="high">{t("riskEngine.high")}</option>
              <option value="critical">{t("riskEngine.critical")}</option>
            </select>
          </div>
          <div className="space-y-2">
            {filteredTimeline.map(ev => {
              const lvl = getLevel(ev.score);
              return (
                <div key={ev.id} className="flex items-center justify-between rounded-lg border p-3 dark:border-gray-700">
                  <div className="flex items-center gap-3">
                    <div className={`flex h-8 w-8 items-center justify-center rounded-lg ${lvl.bg}`}>
                      {lvl.level === "low" ? <CheckCircle2 className={`h-4 w-4 ${lvl.color}`} /> :
                       lvl.level === "critical" ? <Ban className={`h-4 w-4 ${lvl.color}`} /> :
                       <AlertTriangle className={`h-4 w-4 ${lvl.color}`} />}
                    </div>
                    <div>
                      <div className="flex items-center gap-2">
                        <span className="text-sm font-medium">{ev.username}</span>
                        <span className={`px-1.5 py-0.5 rounded text-xs font-bold ${lvl.bg} ${lvl.color}`}>{ev.score}</span>
                        <span className="text-xs text-gray-400">→ {ev.decision}</span>
                      </div>
                      <p className="text-xs text-gray-400">{ev.signals} {t("riskEngine.signalsTriggered")} · {new Date(ev.time).toLocaleTimeString()}</p>
                    </div>
                  </div>
                  <ChevronRight className="h-4 w-4 text-gray-300" />
                </div>
              );
            })}
            {filteredTimeline.length === 0 && <div className={card}><div className="py-8 text-center"><Clock className="mx-auto h-10 w-10 text-gray-300" /><p className="mt-3 text-sm text-gray-400">{t("riskEngine.noEvents")}</p></div></div>}
          </div>
        </div>
      )}

      </>)}
    </div>
  );
}
