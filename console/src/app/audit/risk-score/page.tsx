"use client";

import { useState, useEffect, useCallback } from "react";
import { useApi } from "@/lib/api";
import {
  Gauge, TrendingDown, TrendingUp, AlertTriangle, ShieldCheck,
  Loader2, Lightbulb, Activity, Lock, UserPlus,
} from "lucide-react";
import { useTranslations } from "@/lib/i18n";

interface RiskData {
  current_score: number;
  trend: "up" | "down" | "stable";
  trend_delta: number;
  history: { timestamp: string; score: number }[];
  factors: { name: string; severity: "critical" | "high" | "medium" | "low"; score: number; description: string }[];
  recommendations: { title: string; impact: string; action: string }[];
}

const SEVERITY_COLOR = {
  critical: { bg: "bg-red-100 dark:bg-red-900/30", text: "text-red-700 dark:text-red-400", bar: "bg-red-500" },
  high: { bg: "bg-orange-100 dark:bg-orange-900/30", text: "text-orange-700 dark:text-orange-400", bar: "bg-orange-500" },
  medium: { bg: "bg-yellow-100 dark:bg-yellow-900/30", text: "text-yellow-700 dark:text-yellow-400", bar: "bg-yellow-500" },
  low: { bg: "bg-blue-100 dark:bg-blue-900/30", text: "text-blue-700 dark:text-blue-400", bar: "bg-blue-500" },
};

const SCORE_COLOR = (score: number) =>
  score >= 80 ? "text-green-600" : score >= 60 ? "text-yellow-600" : score >= 40 ? "text-orange-600" : "text-red-600";

const SCORE_GRADE = (score: number) =>
  score >= 90 ? "A" : score >= 80 ? "B" : score >= 70 ? "C" : score >= 60 ? "D" : "F";

export default function RiskScorePage() {
  const t = useTranslations();

  const { apiFetch } = useApi();
  const [data, setData] = useState<RiskData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await apiFetch<RiskData>("/api/v1/audit/risk-score").catch(() => null);
      if (res) setData(res);
    } catch {
      setError("Failed to load risk score data");
    } finally {
      setLoading(false);
    }
  }, [apiFetch]);

  useEffect(() => { load(); const i = setInterval(load, 60000); return () => clearInterval(i); }, [load]);

  const cardCls = "rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-gray-700 dark:bg-gray-800";

  if (loading && !data) {
    return <div className="flex justify-center py-24"><Loader2 className="h-8 w-8 animate-spin text-indigo-600" /></div>;
  }
  if (error || !data) {
    return (
      <div className={cardCls}>
        <div className="py-12 text-center">
          <AlertTriangle className="mx-auto h-12 w-12 text-gray-300" />
          <p className="mt-4 text-sm text-gray-400">{error ?? "No risk data available."}</p>
        </div>
      </div>
    );
  }

  // Simple SVG sparkline
  const maxScore = 100;
  const w = 400, h = 80, pad = 8;
  const pts = data.history.length > 0 ? data.history : [{ timestamp: "", score: data.current_score }];
  const stepX = (w - pad * 2) / Math.max(pts.length - 1, 1);
  const sparkPath = pts.map((p, i) => {
    const x = pad + i * stepX;
    const y = h - pad - (p.score / maxScore) * (h - pad * 2);
    return `${i === 0 ? "M" : "L"} ${x.toFixed(1)} ${y.toFixed(1)}`;
  }).join(" ");

  return (
    <div className="space-y-6">
      <div>
        <h1 className="flex items-center gap-2 text-2xl font-bold text-gray-900 dark:text-white">
          <Gauge className="h-6 w-6 text-indigo-600" /> Risk Score
        </h1>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
          Real-time security posture assessment based on audit signals.
        </p>
      </div>

      <div className="grid gap-6 lg:grid-cols-3">
        {/* Current score gauge */}
        <div className={cardCls}>
          <h3 className="mb-4 flex items-center gap-2 text-sm font-semibold text-gray-700 dark:text-gray-300">
            <ShieldCheck className="h-4 w-4" /> Current Score
          </h3>
          <div className="flex items-center gap-4">
            <div className={`text-5xl font-bold ${SCORE_COLOR(data.current_score)}`}>
              {data.current_score}
            </div>
            <div className="space-y-1">
              <span className="inline-flex items-center gap-1 rounded-full bg-indigo-100 px-2 py-0.5 text-xs font-medium text-indigo-700 dark:bg-indigo-900/30 dark:text-indigo-400">
                Grade {SCORE_GRADE(data.current_score)}
              </span>
              <div className={`flex items-center gap-1 text-sm ${data.trend === "up" ? "text-green-600" : data.trend === "down" ? "text-red-600" : "text-gray-400"}`}>
                {data.trend === "up" ? <TrendingUp className="h-4 w-4" /> : data.trend === "down" ? <TrendingDown className="h-4 w-4" /> : <Activity className="h-4 w-4" />}
                {data.trend === "stable" ? "Stable" : `${data.trend === "up" ? "+" : ""}${data.trend_delta}`}
              </div>
            </div>
          </div>
          <div className="mt-4 h-2 overflow-hidden rounded-full bg-gray-200 dark:bg-gray-700">
            <div className={`h-full rounded-full transition-all ${data.current_score >= 80 ? "bg-green-500" : data.current_score >= 60 ? "bg-yellow-500" : data.current_score >= 40 ? "bg-orange-500" : "bg-red-500"}`} style={{ width: `${data.current_score}%` }} />
          </div>
        </div>

        {/* Trend chart */}
        <div className={`${cardCls} lg:col-span-2`}>
          <h3 className="mb-4 flex items-center gap-2 text-sm font-semibold text-gray-700 dark:text-gray-300">
            <Activity className="h-4 w-4" /> Score History (30 days)
          </h3>
          <svg viewBox={`0 0 ${w} ${h}`} className="w-full">
            <defs>
              <linearGradient id="riskGrad" x1="0" y1="0" x2="0" y2="1">
                <stop offset="0%" stopColor="rgb(99 102 241)" stopOpacity="0.3" />
                <stop offset="100%" stopColor="rgb(99 102 241)" stopOpacity="0" />
              </linearGradient>
            </defs>
            {/* Area fill */}
            <path d={`${sparkPath} L ${w - pad} ${h - pad} L ${pad} ${h - pad} Z`} fill="url(#riskGrad)" />
            {/* Line */}
            <path d={sparkPath} fill="none" stroke="rgb(99 102 241)" strokeWidth="2" strokeLinejoin="round" />
          </svg>
          <div className="mt-2 flex justify-between text-xs text-gray-400">
            <span>{pts.length > 0 ? new Date(pts[0].timestamp).toLocaleDateString() : "—"}</span>
            <span>{pts.length > 0 ? new Date(pts[pts.length - 1].timestamp).toLocaleDateString() : "—"}</span>
          </div>
        </div>
      </div>

      <div className="grid gap-6 lg:grid-cols-2">
        {/* Risk factors breakdown */}
        <div className={cardCls}>
          <h3 className="mb-4 flex items-center gap-2 text-sm font-semibold text-gray-700 dark:text-gray-300">
            <AlertTriangle className="h-4 w-4" /> Risk Factors
          </h3>
          {data.factors.length === 0 ? (
            <p className="py-6 text-center text-sm text-gray-400">No significant risk factors detected.</p>
          ) : (
            <div className="space-y-3">
              {data.factors.map((f, i) => {
                const colors = SEVERITY_COLOR[f.severity];
                return (
                  <div key={i} className="space-y-1">
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-2">
                        <span className={`rounded-full px-2 py-0.5 text-xs font-medium ${colors.bg} ${colors.text}`}>{f.severity}</span>
                        <span className="text-sm font-medium text-gray-700 dark:text-gray-300">{f.name}</span>
                      </div>
                      <span className="text-sm font-bold text-gray-600 dark:text-gray-400">-{f.score}</span>
                    </div>
                    <div className="h-1.5 overflow-hidden rounded-full bg-gray-200 dark:bg-gray-700">
                      <div className={`h-full rounded-full ${colors.bar}`} style={{ width: `${Math.min(f.score * 5, 100)}%` }} />
                    </div>
                    <p className="text-xs text-gray-400">{f.description}</p>
                  </div>
                );
              })}
            </div>
          )}
        </div>

        {/* Recommendations */}
        <div className={cardCls}>
          <h3 className="mb-4 flex items-center gap-2 text-sm font-semibold text-gray-700 dark:text-gray-300">
            <Lightbulb className="h-4 w-4 text-yellow-500" /> Recommended Actions
          </h3>
          {data.recommendations.length === 0 ? (
            <p className="py-6 text-center text-sm text-gray-400">No recommendations at this time.</p>
          ) : (
            <div className="space-y-3">
              {data.recommendations.map((r, i) => (
                <div key={i} className="rounded-lg border border-gray-200 p-3 dark:border-gray-700">
                  <div className="flex items-center justify-between">
                    <span className="text-sm font-medium text-gray-700 dark:text-gray-300">{r.title}</span>
                    <span className="rounded-full bg-green-100 px-2 py-0.5 text-xs font-medium text-green-700 dark:bg-green-900/30 dark:text-green-400">+{r.impact}</span>
                  </div>
                  <p className="mt-1 flex items-start gap-1 text-xs text-gray-400">
                    {r.action.includes("MFA") || r.action.includes("password") ? <Lock className="mt-0.5 h-3 w-3 shrink-0" /> : <UserPlus className="mt-0.5 h-3 w-3 shrink-0" />}
                    {r.action}
                  </p>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
