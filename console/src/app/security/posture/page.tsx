"use client";

import React, { useState, useCallback, useEffect } from "react";
import {
  ShieldCheck, Loader2, AlertCircle, X, TrendingUp, KeyRound, Users,
  Lightbulb, ArrowRight, Smartphone, Lock, AlertTriangle, Activity,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";
import { authHeader } from "@/lib/auth-helpers";

interface Recommendation {
  id: string;
  category: string;
  title: string;
  description: string;
  severity: "low" | "medium" | "high" | "critical";
  impact: number;
  action_url: string;
}

/** Legacy posture data from /api/v1/audit/security-posture */
interface Posture {
  score: number;
  grade: string;
  mfa_adoption_pct: number;
  weak_password_count: number;
  total_users: number;
  active_sessions: number;
  expired_sessions: number;
  failed_logins_24h: number;
  recommendations: Recommendation[];
  last_calculated: string;
}

/** Zero Trust posture data from /api/v1/zt/posture */
interface ZTPosture {
  device_trust_coverage_pct?: number;
  mfa_coverage_pct?: number;
  session_binding_rate_pct?: number;
  unaddressed_critical?: number;
  unaddressed_high?: number;
  zt_score?: number;
  zt_grade?: string;
  policy_violations_24h?: number;
  trusted_devices?: number;
  total_devices?: number;
  // 5-dimension breakdown (NIST 800-207)
  dimensions?: {
    identity: number;
    device: number;
    network: number;
    data: number;
    workload: number;
  };
  // Detailed findings
  findings?: {
    id: string;
    dimension: string;
    title: string;
    severity: "low" | "medium" | "high" | "critical";
    status: "open" | "addressed";
  }[];
  // Allow legacy fields to merge
  score?: number;
  grade?: string;
  mfa_adoption_pct?: number;
  weak_password_count?: number;
  total_users?: number;
  active_sessions?: number;
  expired_sessions?: number;
  failed_logins_24h?: number;
  recommendations?: Recommendation[];
  last_calculated?: string;
}

interface PostureHistoryEntry {
  timestamp: string;
  score: number;
  grade?: string;
}

interface PostureHistory {
  entries?: PostureHistoryEntry[];
  trend?: "improving" | "declining" | "stable";
}

const TENANT_ID = localStorage.getItem("ggid_tenant_id") || "";

const sevColors: Record<string, string> = {
  critical: "text-red-600 bg-red-100 dark:bg-red-900/30 dark:text-red-400",
  high: "text-orange-600 bg-orange-100 dark:bg-orange-900/30 dark:text-orange-400",
  medium: "text-yellow-600 bg-yellow-100 dark:bg-yellow-900/30 dark:text-yellow-400",
  low: "text-blue-600 bg-blue-100 dark:bg-blue-900/30 dark:text-blue-400",
};

function PostureGauge({ score, grade }: { score: number; grade: string }) {
  const clamped = Math.min(100, Math.max(0, score));
  const color = clamped >= 80 ? "#16a34a" : clamped >= 60 ? "#eab308" : clamped >= 40 ? "#f97316" : "#dc2626";
  const gradeColor = grade === "A" ? "text-green-600" : grade === "B" ? "text-lime-600" : grade === "C" ? "text-yellow-600" : grade === "D" ? "text-orange-600" : "text-red-600";
  return (
    <div className="relative flex flex-col items-center">
      <svg width="160" height="90" viewBox="0 0 160 90" aria-hidden="true">
        <path d="M 10 80 A 70 70 0 0 1 150 80" fill="none" stroke="#e5e7eb" strokeWidth="10" strokeLinecap="round" />
        <path d="M 10 80 A 70 70 0 0 1 150 80" fill="none" stroke={color} strokeWidth="10" strokeLinecap="round" strokeDasharray={`${(clamped / 100) * 220} 220`} />
      </svg>
      <div className="-mt-8 flex flex-col items-center">
        <div className="text-3xl font-bold" style={{ color }}>{clamped}</div>
        <div className={`text-2xl font-bold ${gradeColor}`}>{grade}</div>
      </div>
    </div>
  );
}

function MetricBar({ value, color }: { value: number; color: string }) {
  return (
    <div className="h-3 overflow-hidden rounded-full bg-gray-200 dark:bg-gray-700">
      <div className={"h-full rounded-full transition-all " + color} style={{ width: `${Math.min(100, Math.max(0, value))}%` }} />
    </div>
  );
}

/** 5-dimension radar chart (SVG, no dependency) */
function PostureRadar({ dimensions }: { dimensions: { identity: number; device: number; network: number; data: number; workload: number } }) {
  const dims = [
    { label: "Identity", value: dimensions.identity, angle: -90 },
    { label: "Device", value: dimensions.device, angle: -18 },
    { label: "Network", value: dimensions.network, angle: 54 },
    { label: "Data", value: dimensions.data, angle: 126 },
    { label: "Workload", value: dimensions.workload, angle: 198 },
  ];
  const cx = 110, cy = 110, maxR = 80;
  const points = dims.map(d => {
    const rad = (d.angle * Math.PI) / 180;
    const r = (d.value / 100) * maxR;
    return { x: cx + r * Math.cos(rad), y: cy + r * Math.sin(rad), label: d.label, value: d.value, lx: cx + (maxR + 18) * Math.cos(rad), ly: cy + (maxR + 18) * Math.sin(rad) };
  });
  const polygon = points.map(p => `${p.x},${p.y}`).join(" ");

  return (
    <svg width="220" height="220" viewBox="0 0 220 220" className="mx-auto">
      {/* Grid rings */}
      {[20, 40, 60, 80, 100].map(pct => {
        const r = (pct / 100) * maxR;
        const gridPts = dims.map(d => {
          const rad = (d.angle * Math.PI) / 180;
          return `${cx + r * Math.cos(rad)},${cy + r * Math.sin(rad)}`;
        }).join(" ");
        return <polygon key={pct} points={gridPts} fill="none" stroke="#e5e7eb" strokeWidth="0.5" className="dark:stroke-gray-700" />;
      })}
      {/* Axis lines */}
      {dims.map((d, i) => {
        const rad = (d.angle * Math.PI) / 180;
        return <line key={i} x1={cx} y1={cy} x2={cx + maxR * Math.cos(rad)} y2={cy + maxR * Math.sin(rad)} stroke="#e5e7eb" strokeWidth="0.5" className="dark:stroke-gray-700" />;
      })}
      {/* Data polygon */}
      <polygon points={polygon} fill="rgba(16,185,129,0.15)" stroke="#10b981" strokeWidth="2" />
      {/* Data points */}
      {points.map((p, i) => <circle key={i} cx={p.x} cy={p.y} r="3" fill="#10b981" />)}
      {/* Labels */}
      {points.map((p, i) => (
        <text key={i} x={p.lx} y={p.ly} textAnchor={p.lx < cx - 5 ? "end" : p.lx > cx + 5 ? "start" : "middle"}
          dy={p.ly < cy - 5 ? "-2" : p.ly > cy + 5 ? "8" : "4"}
          className="fill-gray-500 dark:fill-gray-400" fontSize="10" fontWeight="600">
          {p.label} ({p.value})
        </text>
      ))}
    </svg>
  );
}

/** Trend sparkline chart (SVG, no dependency) */
function PostureTrendChart({ history }: { history: PostureHistoryEntry[] }) {
  if (history.length < 2) {
    return <p className="py-8 text-center text-sm text-gray-400">Insufficient data for trend analysis</p>;
  }
  const w = 500, h = 120, pad = 20;
  const scores = history.map(e => e.score);
  const maxScore = Math.max(...scores, 100);
  const minScore = Math.min(...scores, 0);
  const range = maxScore - minScore || 1;
  const stepX = (w - pad * 2) / (history.length - 1);

  const pathData = history.map((e, i) => {
    const x = pad + i * stepX;
    const y = h - pad - ((e.score - minScore) / range) * (h - pad * 2);
    return `${i === 0 ? "M" : "L"} ${x} ${y}`;
  }).join(" ");

  const lastScore = scores[scores.length - 1];
  const firstScore = scores[0];
  const trend = lastScore > firstScore + 2 ? "improving" : lastScore < firstScore - 2 ? "declining" : "stable";
  const trendColor = trend === "improving" ? "text-green-600" : trend === "declining" ? "text-red-600" : "text-gray-500";
  const trendIcon = trend === "improving" ? "↑" : trend === "declining" ? "↓" : "→";

  return (
    <div>
      <div className="mb-2 flex items-center justify-between">
        <span className="text-xs text-gray-400">{history.length} data points</span>
        <span className={`text-sm font-medium ${trendColor}`}>{trendIcon} {trend}</span>
      </div>
      <svg width="100%" height={h} viewBox={`0 0 ${w} ${h}`} className="overflow-visible">
        {/* Grid lines */}
        {[0, 25, 50, 75, 100].map(v => {
          const y = h - pad - ((v - minScore) / range) * (h - pad * 2);
          return (
            <g key={v}>
              <line x1={pad} y1={y} x2={w - pad} y2={y} stroke="#e5e7eb" strokeWidth="0.5" className="dark:stroke-gray-700" />
              <text x={pad - 4} y={y + 3} textAnchor="end" className="fill-gray-400" fontSize="9">{v}</text>
            </g>
          );
        })}
        {/* Trend line */}
        <path d={pathData} fill="none" stroke="#10b981" strokeWidth="2" />
        {/* Points */}
        {history.map((e, i) => {
          const x = pad + i * stepX;
          const y = h - pad - ((e.score - minScore) / range) * (h - pad * 2);
          return <circle key={i} cx={x} cy={y} r="2.5" fill="#10b981" />;
        })}
      </svg>
    </div>
  );
}

export default function SecurityPosturePage() {
  const t = useTranslations();
  const [posture, setPosture] = useState<Posture | null>(null);
  const [ztPosture, setZtPosture] = useState<ZTPosture | null>(null);
  const [history, setHistory] = useState<PostureHistoryEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const headers = { ...authHeader(), "X-Tenant-ID": TENANT_ID };

      // Try Zero Trust posture endpoint first
      const ztRes = await fetch("/api/v1/zt/posture", { headers }).catch(() => null);
      if (ztRes?.ok) {
        setZtPosture(await ztRes.json());
      }

      // Also fetch legacy posture for backwards compat
      const res = await fetch("/api/v1/audit/security-posture", { headers }).catch(() => null);
      if (res?.ok) {
        setPosture(await res.json());
      } else if (!ztRes?.ok) {
        // Both failed — show empty state
        setPosture(null);
        setZtPosture(null);
      }

      // Fetch posture history for trend chart
      const histRes = await fetch("/api/v1/audit/security-posture/history?limit=30", { headers }).catch(() => null);
      if (histRes?.ok) {
        const histData = await histRes.json();
        const entries = Array.isArray(histData) ? histData : (histData?.entries || []);
        setHistory(entries);
      }
    } catch {
      setError("Failed to load security posture");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  // Merge ZT and legacy data for display
  const score = ztPosture?.zt_score ?? posture?.score ?? 0;
  const grade = ztPosture?.zt_grade ?? posture?.grade ?? "—";
  const mfaPct = ztPosture?.mfa_coverage_pct ?? posture?.mfa_adoption_pct ?? 0;
  const deviceTrustPct = ztPosture?.device_trust_coverage_pct ?? 0;
  const sessionBindingPct = ztPosture?.session_binding_rate_pct ?? 0;
  const criticalCount = ztPosture?.unaddressed_critical ?? 0;
  const highCount = ztPosture?.unaddressed_high ?? 0;
  const recommendations = posture?.recommendations ?? [];
  const hasData = posture || ztPosture;

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white">
            <ShieldCheck className="h-6 w-6 text-emerald-600" />
            {t("securityPosture.title") || "Zero Trust Security Posture"}
          </h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
            Aggregate view: device trust coverage, MFA adoption, session binding, and unaddressed findings.
          </p>
        </div>
        <button onClick={fetchData} disabled={loading} aria-label="Refresh posture" className="flex items-center gap-2 rounded-lg border border-gray-300 px-3 py-2 text-sm text-gray-600 hover:bg-gray-50 dark:border-gray-700 dark:text-gray-300 dark:hover:bg-gray-800">
          <Activity className={"h-4 w-4 " + (loading ? "animate-spin" : "")} /> Refresh
        </button>
      </div>

      {/* Error */}
      {error && (
        <div role="alert" className="flex items-center gap-2 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-400">
          <AlertCircle className="h-4 w-4 shrink-0" />{error}
          <button onClick={() => setError(null)} aria-label="Dismiss error" className="ml-auto"><X className="h-4 w-4" /></button>
        </div>
      )}

      {loading ? (
        <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin text-emerald-600" /></div>
      ) : hasData ? (
        <>
          {/* Top row: Score + 4 ZT metrics */}
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
            {/* ZT Score gauge */}
            <div className={`${cardCls} flex flex-col items-center justify-center`}>
              <PostureGauge score={score} grade={grade} />
              <div className="mt-2 text-xs uppercase text-gray-400">ZT Posture Score</div>
            </div>

            {/* Device Trust Coverage */}
            <div className={cardCls}>
              <div className="mb-3 flex items-center gap-2">
                <Smartphone className="h-4 w-4 text-blue-500" />
                <span className="text-xs font-semibold uppercase text-gray-400">Device Trust</span>
              </div>
              <div className="text-3xl font-bold text-blue-600">{deviceTrustPct}%</div>
              <div className="mt-2"><MetricBar value={deviceTrustPct} color="bg-blue-500" /></div>
              {ztPosture?.trusted_devices !== undefined && (
                <p className="mt-2 text-xs text-gray-400">
                  {ztPosture.trusted_devices} / {ztPosture.total_devices ?? 0} devices trusted
                </p>
              )}
            </div>

            {/* MFA Coverage */}
            <div className={cardCls}>
              <div className="mb-3 flex items-center gap-2">
                <Users className="h-4 w-4 text-indigo-500" />
                <span className="text-xs font-semibold uppercase text-gray-400">MFA Coverage</span>
              </div>
              <div className="text-3xl font-bold text-indigo-600">{mfaPct}%</div>
              <div className="mt-2"><MetricBar value={mfaPct} color="bg-indigo-500" /></div>
              {posture && (
                <p className="mt-2 flex items-center gap-1 text-xs text-gray-400">
                  <KeyRound className="h-3 w-3" />
                  {posture.weak_password_count} weak / {posture.total_users} users
                </p>
              )}
            </div>

            {/* Session Binding Rate */}
            <div className={cardCls}>
              <div className="mb-3 flex items-center gap-2">
                <Lock className="h-4 w-4 text-purple-500" />
                <span className="text-xs font-semibold uppercase text-gray-400">Session Binding</span>
              </div>
              <div className="text-3xl font-bold text-purple-600">{sessionBindingPct}%</div>
              <div className="mt-2"><MetricBar value={sessionBindingPct} color="bg-purple-500" /></div>
              {posture && (
                <p className="mt-2 text-xs text-gray-400">
                  {posture.active_sessions} active / {posture.expired_sessions} expired
                </p>
              )}
            </div>
          </div>

          {/* Second row: Radar chart + Trend chart */}
          <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
            {/* 5-dimension radar chart */}
            <div className={cardCls}>
              <div className="mb-4 flex items-center gap-2">
                <Activity className="h-4 w-4 text-emerald-500" />
                <span className="text-xs font-semibold uppercase text-gray-400">NIST 800-207 Posture Dimensions</span>
              </div>
              {ztPosture?.dimensions ? (
                <PostureRadar dimensions={ztPosture.dimensions} />
              ) : (
                <div className="py-8 text-center">
                  <p className="text-sm text-gray-400">Dimension breakdown not available</p>
                  <p className="mt-1 text-xs text-gray-400">Requires ZT posture data with 5-dimension scoring</p>
                </div>
              )}
            </div>

            {/* Historical trend chart */}
            <div className={cardCls}>
              <div className="mb-4 flex items-center gap-2">
                <TrendingUp className="h-4 w-4 text-emerald-500" />
                <span className="text-xs font-semibold uppercase text-gray-400">Score History (30 days)</span>
              </div>
              <PostureTrendChart history={history} />
            </div>
          </div>

          {/* Third row: Critical findings + Session stats */}
          <div className="grid grid-cols-1 gap-4 lg:grid-cols-3">
            {/* Unaddressed findings */}
            <div className={`${cardCls} lg:col-span-2`}>
              <div className="mb-4 flex items-center gap-2">
                <AlertTriangle className="h-4 w-4 text-amber-500" />
                <span className="text-xs font-semibold uppercase text-gray-400">Unaddressed Findings</span>
              </div>
              <div className="grid grid-cols-2 gap-4 sm:grid-cols-4">
                <div className="rounded-lg border border-red-200 p-3 dark:border-red-900/50">
                  <p className="text-xs font-medium uppercase text-gray-400">Critical</p>
                  <p className={"mt-1 text-2xl font-bold " + (criticalCount > 0 ? "text-red-600" : "text-gray-300")}>{criticalCount}</p>
                </div>
                <div className="rounded-lg border border-orange-200 p-3 dark:border-orange-900/50">
                  <p className="text-xs font-medium uppercase text-gray-400">High</p>
                  <p className={"mt-1 text-2xl font-bold " + (highCount > 0 ? "text-orange-600" : "text-gray-300")}>{highCount}</p>
                </div>
                <div className="rounded-lg border border-gray-200 p-3 dark:border-gray-700">
                  <p className="text-xs font-medium uppercase text-gray-400">Failed Logins 24h</p>
                  <p className="mt-1 text-2xl font-bold text-gray-600 dark:text-gray-300">{posture?.failed_logins_24h ?? 0}</p>
                </div>
                <div className="rounded-lg border border-gray-200 p-3 dark:border-gray-700">
                  <p className="text-xs font-medium uppercase text-gray-400">Policy Violations 24h</p>
                  <p className="mt-1 text-2xl font-bold text-gray-600 dark:text-gray-300">{ztPosture?.policy_violations_24h ?? 0}</p>
                </div>
              </div>
            </div>

            {/* Last calculated */}
            <div className={cardCls}>
              <div className="mb-3 flex items-center gap-2">
                <TrendingUp className="h-4 w-4 text-gray-400" />
                <span className="text-xs font-semibold uppercase text-gray-400">Last Updated</span>
              </div>
              <p className="text-sm text-gray-600 dark:text-gray-400">
                {posture?.last_calculated || ztPosture?.last_calculated
                  ? new Date(posture?.last_calculated || ztPosture?.last_calculated || "").toLocaleString()
                  : "Real-time"}
              </p>
            </div>
          </div>

          {/* Recommendations */}
          <div>
            <h2 className="mb-3 flex items-center gap-2 text-sm font-semibold uppercase text-gray-500">
              <Lightbulb className="h-4 w-4" /> Recommendations
            </h2>
            {recommendations.length === 0 ? (
              <div className={cardCls}>
                <div className="py-8 text-center">
                  <ShieldCheck className="mx-auto h-10 w-10 text-green-300" />
                  <p className="mt-3 text-sm text-gray-400">No recommendations. Your posture is excellent.</p>
                </div>
              </div>
            ) : (
              <div className="space-y-2">
                {recommendations.map((r: any) => (
                  <div key={r.id} className={`${cardCls} flex items-center justify-between py-3`}>
                    <div className="flex items-center gap-3">
                      <span className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${sevColors[r.severity] || ""}`}>{r.severity}</span>
                      <div>
                        <div className="font-medium text-gray-900 dark:text-white">{r.title}</div>
                        <div className="text-xs text-gray-400">{r.description}</div>
                      </div>
                    </div>
                    <div className="flex items-center gap-3">
                      <span className="text-xs text-gray-400">+{r.impact} pts</span>
                      <ArrowRight className="h-4 w-4 text-gray-300" />
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        </>
      ) : (
        <div className={cardCls}>
          <div className="py-12 text-center">
            <ShieldCheck className="mx-auto h-12 w-12 text-gray-300" />
            <p className="mt-4 text-sm text-gray-400">No posture data available.</p>
            <p className="mt-1 text-xs text-gray-400">
              Posture data is being collected. This feature requires device trust and MFA enrollment data to generate insights.
            </p>
          </div>
        </div>
      )}
    </div>
  );
}
